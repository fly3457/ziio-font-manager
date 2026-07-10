import {memo, type CSSProperties, type MouseEvent as ReactMouseEvent, type PointerEvent as ReactPointerEvent, type UIEvent, useEffect, useRef, useState} from 'react';
import './App.css';
import brandLogo from './assets/images/logo.svg';
import {
    AlertTriangle,
    CheckCircle2,
    Columns3,
    Download,
    ExternalLink,
    FolderOpen,
    FolderPlus,
    Grid3X3,
    HardDriveDownload,
    Info,
    Link2,
    List,
    MoreHorizontal,
    RefreshCw,
    Search,
    Server,
    Settings,
    Star,
    Trash2,
    X
} from 'lucide-react';
import {useTranslation} from 'react-i18next';
import type {TFunction} from 'i18next';
import {EventsOn, LogInfo, LogWarning} from '../wailsjs/runtime/runtime';
import {
    api,
    type AppInfo,
    type FontDetail,
    type FontFolder,
    type FontItem,
    type FontQuery,
    type LibraryRoot,
    type OperationProgress,
    type OperationResult,
    type PreviewResponse
} from './api';
import {DEFAULT_SAMPLE_TEXTS, LANGUAGE_STORAGE_KEY, SUPPORTED_LANGUAGES, type SupportedLanguage} from './i18n';

const PAGE_SIZE = 20;
const PREVIEW_CONCURRENCY = 2;
const PREVIEW_LRU_LIMIT = 1000;
const PREVIEW_ROOT_MARGIN = '900px 0px';
const PREVIEW_SLOW_MS = 300;
const EXPANDED_FOLDERS_KEY = 'ziio.fontManager.expandedFolders.v1';
const LAYOUT_PREFS_KEY = 'ziio.fontManager.layout.v1';
const CARD_COLUMN_OPTIONS = [2, 3, 4, 5] as const;
const DEFAULT_LAYOUT_PREFS = {sidebarWidth: 250, detailWidth: 320};
const SIDEBAR_MIN = 210;
const SIDEBAR_MAX = 420;
const DETAIL_MIN = 280;
const DETAIL_MAX = 520;
const MIDDLE_MIN = 440;
const RESIZE_HANDLE_TOTAL = 12;

type ViewMode = 'list' | 'grid';
type CardColumnCount = typeof CARD_COLUMN_OPTIONS[number];
type PreviewQueueItem = {faceId: number; generation: number; sampleText: string};
type LayoutPrefs = {sidebarWidth: number; detailWidth: number};
type ActionMenuState =
    | {type: 'root'; root: LibraryRoot; key: string; left: number; top: number}
    | {type: 'folder'; root: LibraryRoot; folder: FontFolder; key: string; left: number; top: number};

function App() {
    const {t, i18n} = useTranslation();
    const [roots, setRoots] = useState<LibraryRoot[]>([]);
    const [folders, setFolders] = useState<FontFolder[]>([]);
    const [fonts, setFonts] = useState<FontItem[]>([]);
    const [selectedRoot, setSelectedRoot] = useState(0);
    const [selectedFolder, setSelectedFolder] = useState('');
    const [expandedFolderKeys, setExpandedFolderKeys] = useState<string[]>(loadExpandedFolderKeys);
    const [selectedFace, setSelectedFace] = useState<number | null>(null);
    const [checkedFaces, setCheckedFaces] = useState<number[]>([]);
    const [detail, setDetail] = useState<FontDetail | null>(null);
    const [previews, setPreviews] = useState<Record<number, PreviewResponse>>({});
    const [previewLoadingIds, setPreviewLoadingIds] = useState<Record<number, boolean>>({});
    const [previewFailures, setPreviewFailures] = useState<Record<number, boolean>>({});
    const [query, setQuery] = useState('');
    const [favoritesOnly, setFavoritesOnly] = useState(false);
    const [installedOnly, setInstalledOnly] = useState(false);
    const [viewMode, setViewMode] = useState<ViewMode>('grid');
    const [cardColumns, setCardColumns] = useState<CardColumnCount>(3);
    const [installScope, setInstallScope] = useState<'user' | 'machine'>('user');
    const [sampleText, setSampleText] = useState(() => t('preview.sample'));
    const [fontSize, setFontSize] = useState(34);
    const [busy, setBusy] = useState(false);
    const [loadingMore, setLoadingMore] = useState(false);
    const [hasMore, setHasMore] = useState(false);
    const [notice, setNotice] = useState<{type: 'info' | 'error' | 'success', text: string} | null>(null);
    const [operationProgress, setOperationProgress] = useState<OperationProgress | null>(null);
    const [layoutPrefs, setLayoutPrefs] = useState<LayoutPrefs>(loadLayoutPrefs);
    const [actionMenu, setActionMenu] = useState<ActionMenuState | null>(null);
    const [settingsOpen, setSettingsOpen] = useState(false);
    const [appInfo, setAppInfo] = useState<AppInfo | null>(null);
    const [collapsedRootIds, setCollapsedRootIds] = useState<number[]>([]);
    const [gridColumnMenuOpen, setGridColumnMenuOpen] = useState(false);
    const [gridColumnMenuSuppressedUntilLeave, setGridColumnMenuSuppressedUntilLeave] = useState(false);

    const activeIds = checkedFaces.length > 0 ? checkedFaces : selectedFace ? [selectedFace] : [];
    const activeRoot = roots.find(root => root.id === selectedRoot);
    const userRoots = roots.filter(root => root.kind !== 'system');
    const systemRoots = roots.filter(root => root.kind === 'system');
    const anyScanning = roots.some(root => root.scanStatus === 'running');
    const resultsRef = useRef<HTMLElement | null>(null);
    const actionMenuRef = useRef<HTMLDivElement | null>(null);
    const previewQueue = useRef<PreviewQueueItem[]>([]);
    const queuedPreviewIds = useRef<Set<string>>(new Set());
    const inFlightPreviewIds = useRef<Set<string>>(new Set());
    const resolvedPreviewIds = useRef<Set<number>>(new Set());
    const failedPreviewIds = useRef<Set<number>>(new Set());
    const visiblePreviewIds = useRef<Set<number>>(new Set());
    const loadedFontFaces = useRef<Map<number, FontFace>>(new Map());
    const previewAccessTimes = useRef<Map<number, number>>(new Map());
    const fontById = useRef<Map<number, FontItem>>(new Map());
    const selectedFaceRef = useRef<number | null>(null);
    const pendingPreviewUpdates = useRef<Record<number, PreviewResponse>>({});
    const pendingFailureUpdates = useRef<Record<number, boolean>>({});
    const pendingSettledPreviewIds = useRef<Set<number>>(new Set());
    const previewFlushTimer = useRef<number | null>(null);
    const previewGeneration = useRef(0);
    const sampleTextRef = useRef(sampleText);
    sampleTextRef.current = sampleText;

    useEffect(() => {
        loadRoots();
        loadFonts(true);
        api.appInfo()
            .then(info => setAppInfo(info))
            .catch(error => logFrontendWarning(`app info failed error=${error instanceof Error ? error.message : String(error)}`));
    }, []);

    useEffect(() => {
        window.localStorage.setItem(EXPANDED_FOLDERS_KEY, JSON.stringify(expandedFolderKeys));
    }, [expandedFolderKeys]);

    useEffect(() => {
        setSampleText(current => {
            const defaults = Object.values(DEFAULT_SAMPLE_TEXTS);
            return defaults.includes(current) ? t('preview.sample') : current;
        });
    }, [i18n.language, t]);

    useEffect(() => {
        const resize = () => setLayoutPrefs(current => clampLayoutPrefs(current));
        window.addEventListener('resize', resize);
        return () => window.removeEventListener('resize', resize);
    }, []);

    useEffect(() => {
        if (!actionMenu) {
            return;
        }
        const closeOnOutsideClick = (event: PointerEvent) => {
            const target = event.target as Node | null;
            if (target && actionMenuRef.current?.contains(target)) {
                return;
            }
            setActionMenu(null);
        };
        const closeOnEscape = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                setActionMenu(null);
            }
        };
        const closeOnScroll = () => setActionMenu(null);
        window.addEventListener('pointerdown', closeOnOutsideClick);
        window.addEventListener('keydown', closeOnEscape);
        window.addEventListener('scroll', closeOnScroll, true);
        return () => {
            window.removeEventListener('pointerdown', closeOnOutsideClick);
            window.removeEventListener('keydown', closeOnEscape);
            window.removeEventListener('scroll', closeOnScroll, true);
        };
    }, [actionMenu]);

    useEffect(() => {
        const timer = window.setTimeout(() => loadFonts(true), 180);
        return () => window.clearTimeout(timer);
    }, [query, selectedRoot, selectedFolder, favoritesOnly, installedOnly]);

    useEffect(() => {
        if (selectedRoot > 0) {
            loadFolders(selectedRoot);
        } else {
            setFolders([]);
        }
    }, [selectedRoot]);

    useEffect(() => {
        if (!anyScanning) {
            return;
        }
        const timer = window.setInterval(() => {
            loadRoots();
            if (selectedRoot > 0) {
                loadFolders(selectedRoot);
            }
            loadFonts(true);
        }, 2000);
        return () => window.clearInterval(timer);
    }, [anyScanning, selectedRoot, selectedFolder, query, favoritesOnly, installedOnly]);

    useEffect(() => {
        if (!selectedFace) {
            setDetail(null);
            return;
        }
        api.getFontDetail(selectedFace).then(setDetail).catch(showError);
    }, [selectedFace]);

    useEffect(() => {
        fontById.current = new Map(fonts.map(font => [font.faceId, font]));
    }, [fonts]);

    useEffect(() => {
        selectedFaceRef.current = selectedFace;
        if (selectedFace) {
            enqueuePreviewIds([selectedFace], true);
        }
    }, [selectedFace, fonts]);

    useEffect(() => {
        previewGeneration.current += 1;
        clearPreviewRuntimeState();
        setPreviews({});
        setPreviewFailures({});
        setPreviewLoadingIds({});

        const timer = window.setTimeout(() => {
            const visible = Array.from(visiblePreviewIds.current);
            const selected = selectedFaceRef.current;
            enqueuePreviewIds(selected ? [selected, ...visible] : visible, true);
        }, 0);
        return () => window.clearTimeout(timer);
    }, [sampleText]);

    useEffect(() => {
        const stop = EventsOn('font-operation-progress', (progress: OperationProgress) => {
            if (progress?.operation === 'install') {
                setOperationProgress(progress);
            }
        });
        return stop;
    }, []);

    useEffect(() => {
        const root = resultsRef.current;
        if (!root) {
            return;
        }

        visiblePreviewIds.current.clear();
        const observer = new IntersectionObserver(entries => {
            const entered: number[] = [];
            entries.forEach(entry => {
                const faceId = Number((entry.target as HTMLElement).dataset.faceId);
                if (!faceId) {
                    return;
                }
                if (entry.isIntersecting) {
                    visiblePreviewIds.current.add(faceId);
                    previewAccessTimes.current.set(faceId, Date.now());
                    entered.push(faceId);
                } else {
                    visiblePreviewIds.current.delete(faceId);
                }
            });
            if (entered.length > 0) {
                enqueuePreviewIds(entered, false);
                if (entered.length >= 8 || previewQueue.current.length > 20) {
                    logFrontendInfo(`preview visible=${visiblePreviewIds.current.size} queued=${previewQueue.current.length} inFlight=${inFlightPreviewIds.current.size}`);
                }
            }
        }, {
            root,
            rootMargin: PREVIEW_ROOT_MARGIN,
            threshold: 0
        });

        root.querySelectorAll<HTMLElement>('.font-card[data-face-id]').forEach(node => observer.observe(node));
        return () => observer.disconnect();
    }, [fonts, viewMode, cardColumns, sampleText]);

    async function loadRoots() {
        try {
            const next = await api.listRoots();
            setRoots(next ?? []);
        } catch (error) {
            showError(error);
        }
    }

    async function loadFolders(rootId: number) {
        try {
            const next = await api.listFolders(rootId);
            setFolders(next ?? []);
        } catch {
            setFolders([]);
        }
    }

    async function loadFonts(reset: boolean, overrides: Partial<FontQuery> = {}, autoSelectFirst = true) {
        const offset = reset ? 0 : fonts.length;
        try {
            const next = await api.searchFonts({
                query: overrides.query ?? query,
                rootId: overrides.rootId ?? selectedRoot,
                folderPath: overrides.folderPath ?? selectedFolder,
                folderRecursive: true,
                favoritesOnly: overrides.favoritesOnly ?? favoritesOnly,
                installedOnly: overrides.installedOnly ?? installedOnly,
                limit: PAGE_SIZE,
                offset: overrides.offset ?? offset
            });
            setHasMore((next?.length ?? 0) === PAGE_SIZE);
            if (reset) {
                setFonts(next ?? []);
                setCheckedFaces([]);
                setSelectedFace(autoSelectFirst && next?.length ? next[0].faceId : null);
            } else {
                setFonts(current => [...current, ...(next ?? [])]);
            }
        } catch (error) {
            showError(error);
        }
    }

    async function addRoot() {
        await runBusy(async () => {
            const root = await api.chooseAndAddRoot();
            setSelectedRoot(root.id);
            setSelectedFolder('');
            setFavoritesOnly(false);
            setInstalledOnly(false);
            setNotice({type: 'success', text: t('notices.addedRoot', {name: root.name})});
            await loadRoots();
            await loadFolders(root.id);
            await loadFonts(true);
        });
    }

    async function scanSystemFonts() {
        await runBusy(async () => {
            const next = await api.scanSystemFonts();
            setNotice({type: 'success', text: t('notices.scanSystemStarted', {count: next.length})});
            await loadRoots();
            if (next[0]) {
                setSelectedRoot(next[0].id);
                setSelectedFolder('');
            }
        });
    }

    async function rescanAllRoots() {
        if (roots.length === 0) {
            setNotice({type: 'info', text: t('notices.noRoots')});
            return;
        }
        await runBusy(async () => {
            const queued = await api.rescanAllRoots();
            setNotice({
                type: queued > 0 ? 'success' : 'info',
                text: queued > 0 ? t('notices.rescanAllStarted', {count: queued}) : t('notices.allRootsScanning')
            });
            await loadRoots();
            scheduleScanRefresh();
        });
    }

    async function rescanRoot(root: LibraryRoot) {
        if (root.scanStatus === 'running') {
            setNotice({type: 'info', text: t('notices.rootScanningSync')});
            return;
        }
        await runBusy(async () => {
            await api.rescanRoot(root.id);
            setNotice({type: 'success', text: t('notices.syncRootStarted', {name: root.name})});
            await loadRoots();
            if (selectedRoot === root.id) {
                await loadFolders(root.id);
                await loadFonts(true);
            }
            scheduleScanRefresh(root.id);
        });
    }

    async function rescanFolder(root: LibraryRoot, folder: FontFolder) {
        if (root.scanStatus === 'running') {
            setNotice({type: 'info', text: t('notices.rootScanningFolderSync')});
            return;
        }
        await runBusy(async () => {
            await api.rescanFolder(root.id, folder.path);
            setNotice({type: 'success', text: t('notices.syncFolderStarted', {path: folder.path})});
            await loadRoots();
            if (selectedRoot === root.id) {
                await loadFolders(root.id);
                await loadFonts(true);
            }
            scheduleScanRefresh(root.id);
        });
    }

    function scheduleScanRefresh(rootId = selectedRoot) {
        window.setTimeout(() => {
            loadRoots();
            if (rootId > 0) {
                loadFolders(rootId);
            }
            loadFonts(true);
        }, 600);
    }

    function changeLanguage(language: SupportedLanguage) {
        i18n.changeLanguage(language).catch(showError);
        try {
            window.localStorage.setItem(LANGUAGE_STORAGE_KEY, language);
        } catch {
            // localStorage can be unavailable in restricted WebView contexts.
        }
    }

    function openRootActionMenu(root: LibraryRoot, event: ReactMouseEvent<HTMLButtonElement>) {
        event.preventDefault();
        event.stopPropagation();
        setActionMenu({
            type: 'root',
            root,
            key: rootMenuKey(root),
            ...menuPosition(event.currentTarget.getBoundingClientRect(), root.kind === 'system' ? 104 : 148)
        });
    }

    function openFolderActionMenu(root: LibraryRoot, folder: FontFolder, event: ReactMouseEvent<HTMLButtonElement>) {
        event.preventDefault();
        event.stopPropagation();
        setActionMenu({
            type: 'folder',
            root,
            folder,
            key: folderMenuKey(root, folder),
            ...menuPosition(event.currentTarget.getBoundingClientRect(), 96)
        });
    }

    function closeActionMenu() {
        setActionMenu(null);
    }

    function startResizePane(pane: 'sidebar' | 'detail', event: ReactPointerEvent<HTMLDivElement>) {
        if (event.button !== 0) {
            return;
        }
        event.preventDefault();
        const startX = event.clientX;
        const start = layoutPrefs;
        let latest = start;
        document.body.classList.add('resizing-layout');

        const move = (moveEvent: PointerEvent) => {
            const delta = moveEvent.clientX - startX;
            latest = clampLayoutPrefs({
                sidebarWidth: pane === 'sidebar' ? start.sidebarWidth + delta : start.sidebarWidth,
                detailWidth: pane === 'detail' ? start.detailWidth - delta : start.detailWidth
            });
            setLayoutPrefs(latest);
        };
        const stop = () => {
            window.removeEventListener('pointermove', move);
            window.removeEventListener('pointerup', stop);
            document.body.classList.remove('resizing-layout');
            persistLayoutPrefs(latest);
        };

        window.addEventListener('pointermove', move);
        window.addEventListener('pointerup', stop);
    }

    function resetPaneWidth(pane: 'sidebar' | 'detail') {
        const next = clampLayoutPrefs({
            sidebarWidth: pane === 'sidebar' ? DEFAULT_LAYOUT_PREFS.sidebarWidth : layoutPrefs.sidebarWidth,
            detailWidth: pane === 'detail' ? DEFAULT_LAYOUT_PREFS.detailWidth : layoutPrefs.detailWidth
        });
        setLayoutPrefs(next);
        persistLayoutPrefs(next);
    }

    async function removeRoot(root: LibraryRoot) {
        if (root.kind === 'system') {
            setNotice({type: 'error', text: t('notices.systemRootCannotDelete')});
            return;
        }
        if (root.scanStatus === 'running') {
            setNotice({type: 'info', text: t('notices.rootScanningDelete')});
            return;
        }
        if (!window.confirm(t('confirm.removeRoot', {name: root.name, path: root.path}))) {
            return;
        }

        await runBusy(async () => {
            await api.removeRoot(root.id);
            const deletingActiveRoot = selectedRoot === root.id;
            setExpandedFolderKeys(current => current.filter(item => !item.startsWith(`${root.id}:`)));
            setCollapsedRootIds(current => current.filter(id => id !== root.id));
            if (deletingActiveRoot) {
                setSelectedRoot(0);
                setSelectedFolder('');
                setFavoritesOnly(false);
                setInstalledOnly(false);
                setFolders([]);
                setCheckedFaces([]);
                setSelectedFace(null);
                setDetail(null);
            }
            await loadRoots();
            await loadFonts(
                true,
                deletingActiveRoot ? {rootId: 0, folderPath: '', favoritesOnly: false, installedOnly: false} : {},
                !deletingActiveRoot
            );
            setNotice({type: 'success', text: t('notices.removedRoot', {name: root.name})});
        });
    }

    async function loadMore() {
        if (loadingMore || !hasMore) {
            return;
        }
        setLoadingMore(true);
        try {
            await loadFonts(false);
        } finally {
            setLoadingMore(false);
        }
    }

    function handleResultsScroll(event: UIEvent<HTMLElement>) {
        const el = event.currentTarget;
        const remaining = el.scrollHeight - el.scrollTop - el.clientHeight;
        if (remaining < 360) {
            loadMore();
        }
    }

    async function toggleFavorite(font: FontItem) {
        try {
            await api.setFavorite(font.faceId, !font.isFavorite);
            setFonts(current => current.map(item => item.faceId === font.faceId ? {...item, isFavorite: !font.isFavorite} : item));
            if (detail?.faceId === font.faceId) {
                setDetail({...detail, isFavorite: !font.isFavorite});
            }
        } catch (error) {
            showError(error);
        }
    }

    async function install(mode: 'copy' | 'link') {
        if (activeIds.length === 0) {
            setNotice({type: 'info', text: t('notices.chooseFont')});
            return;
        }
        setOperationProgress({
            operation: 'install',
            mode,
            scope: installScope,
            current: 0,
            total: activeIds.length,
            succeeded: 0,
            failed: 0,
            faceId: 0,
            fileId: 0,
            fileName: '',
            status: 'start',
            message: '',
            done: false
        });
        await runBusy(async () => {
            const result = await api.installFonts(activeIds, mode, installScope);
            applyOperationNotice(result);
            await loadFonts(true);
            if (selectedFace) {
                setDetail(await api.getFontDetail(selectedFace));
            }
        });
        window.setTimeout(() => setOperationProgress(null), 900);
    }

    async function uninstall() {
        if (activeIds.length === 0) {
            setNotice({type: 'info', text: t('notices.chooseFont')});
            return;
        }
        if (!window.confirm(t('confirm.uninstall', {count: activeIds.length}))) {
            return;
        }
        await runBusy(async () => {
            const result = await api.uninstallFonts(activeIds, true);
            applyOperationNotice(result);
            await loadFonts(true);
            if (selectedFace) {
                setDetail(await api.getFontDetail(selectedFace));
            }
        });
    }

    async function runBusy(work: () => Promise<void>) {
        setBusy(true);
        try {
            await work();
        } catch (error) {
            showError(error);
        } finally {
            setBusy(false);
        }
    }

    function selectRoot(rootId: number) {
        closeActionMenu();
        if (selectedRoot === rootId) {
            setSelectedFolder('');
            setFavoritesOnly(false);
            setInstalledOnly(false);
            setCollapsedRootIds(current => current.includes(rootId) ? current.filter(id => id !== rootId) : [...current, rootId]);
            return;
        }
        setSelectedRoot(rootId);
        setSelectedFolder('');
        setFavoritesOnly(false);
        setInstalledOnly(false);
        setCollapsedRootIds(current => current.filter(id => id !== rootId));
    }

    function selectFolder(rootId: number, path: string, hasChildren: boolean) {
        closeActionMenu();
        setSelectedRoot(rootId);
        setFavoritesOnly(false);
        setInstalledOnly(false);
        setCollapsedRootIds(current => current.filter(id => id !== rootId));
        setSelectedFolder(path);
        if (hasChildren) {
            const key = folderKey(rootId, path);
            setExpandedFolderKeys(current => current.includes(key) ? current.filter(item => item !== key) : [...current, key]);
        }
    }

    function expandFolder(rootId: number, path: string) {
        const key = folderKey(rootId, path);
        setExpandedFolderKeys(current => current.includes(key) ? current : [...current, key]);
    }

    function toggleFolder(rootId: number, path: string) {
        closeActionMenu();
        setCollapsedRootIds(current => current.filter(id => id !== rootId));
        const key = folderKey(rootId, path);
        setExpandedFolderKeys(current => current.includes(key) ? current.filter(item => item !== key) : [...current, key]);
    }

    function openGridColumnMenu() {
        if (viewMode === 'grid' && !gridColumnMenuSuppressedUntilLeave) {
            setGridColumnMenuOpen(true);
        }
    }

    function closeGridColumnMenuAfterLeave() {
        setGridColumnMenuOpen(false);
        setGridColumnMenuSuppressedUntilLeave(false);
    }

    function activateListView() {
        setViewMode('list');
        setGridColumnMenuOpen(false);
        setGridColumnMenuSuppressedUntilLeave(false);
    }

    function activateGridView() {
        setViewMode('grid');
        setGridColumnMenuOpen(false);
        setGridColumnMenuSuppressedUntilLeave(true);
    }

    function selectCardColumnCount(columns: CardColumnCount) {
        setCardColumns(columns);
        setGridColumnMenuOpen(false);
        setGridColumnMenuSuppressedUntilLeave(true);
    }

    function applyOperationNotice(result: OperationResult) {
        const firstError = result.messages?.find(m => m.level === 'error')?.message;
        if ((result.failed ?? 0) > 0) {
            setNotice({
                type: 'error',
                text: t('notices.operationFailed', {
                    succeeded: result.succeeded,
                    failed: result.failed,
                    message: firstError ? t('notices.errorPrefix', {message: firstError}) : ''
                })
            });
        } else {
            setNotice({type: 'success', text: t('notices.operationCompleted', {count: result.succeeded})});
        }
    }

    function showError(error: unknown) {
        const text = error instanceof Error ? error.message : String(error);
        setNotice({type: 'error', text});
    }

    function renderActionMenu() {
        if (!actionMenu) {
            return null;
        }
        const scanning = actionMenu.root.scanStatus === 'running';
        const disabled = busy || scanning;
        const menuTitle = actionMenu.type === 'root' ? actionMenu.root.name : actionMenu.folder.name;
        return (
            <div
                className="action-menu"
                ref={actionMenuRef}
                role="menu"
                aria-label={t('menu.operations', {name: menuTitle})}
                style={{left: actionMenu.left, top: actionMenu.top}}
            >
                <div className="action-menu-title" title={actionMenu.type === 'root' ? actionMenu.root.path : actionMenu.folder.path}>
                    {menuTitle}
                </div>
                {scanning && <div className="action-menu-hint">{t('menu.scanningHint')}</div>}
                {actionMenu.type === 'root' ? (
                    <>
                        <button
                            role="menuitem"
                            disabled={disabled}
                            onClick={() => {
                                const root = actionMenu.root;
                                closeActionMenu();
                                rescanRoot(root);
                            }}
                        >
                            <RefreshCw size={14}/>
                            <span>{t('actions.syncLibrary')}</span>
                        </button>
                        {actionMenu.root.kind !== 'system' && (
                            <button
                                role="menuitem"
                                className="danger"
                                disabled={disabled}
                                onClick={() => {
                                    const root = actionMenu.root;
                                    closeActionMenu();
                                    removeRoot(root);
                                }}
                            >
                                <Trash2 size={14}/>
                                <span>{t('actions.removeLibrary')}</span>
                            </button>
                        )}
                    </>
                ) : (
                    <button
                        role="menuitem"
                        disabled={disabled}
                        onClick={() => {
                            const root = actionMenu.root;
                            const folder = actionMenu.folder;
                            closeActionMenu();
                            rescanFolder(root, folder);
                        }}
                    >
                        <RefreshCw size={14}/>
                        <span>{t('actions.syncFolder')}</span>
                    </button>
                )}
            </div>
        );
    }

    function toggleChecked(faceId: number) {
        setCheckedFaces(current => current.includes(faceId) ? current.filter(id => id !== faceId) : [...current, faceId]);
    }

    function clearPreviewRuntimeState() {
        if (previewFlushTimer.current !== null) {
            window.clearTimeout(previewFlushTimer.current);
            previewFlushTimer.current = null;
        }
        loadedFontFaces.current.forEach(fontFace => document.fonts.delete(fontFace));
        loadedFontFaces.current.clear();
        previewAccessTimes.current.clear();
        previewQueue.current = [];
        queuedPreviewIds.current.clear();
        inFlightPreviewIds.current.clear();
        resolvedPreviewIds.current.clear();
        failedPreviewIds.current.clear();
        pendingPreviewUpdates.current = {};
        pendingFailureUpdates.current = {};
        pendingSettledPreviewIds.current.clear();
        logFrontendInfo(`preview sample changed generation=${previewGeneration.current}`);
    }

    function previewQueueKey(faceId: number, generation: number) {
        return `${generation}:${faceId}`;
    }

    function enqueuePreviewIds(faceIds: number[], priority: boolean) {
        const generation = previewGeneration.current;
        const previewText = sampleTextRef.current;
        const pending: PreviewQueueItem[] = [];
        faceIds.forEach(faceId => {
            const font = fontById.current.get(faceId);
            const key = previewQueueKey(faceId, generation);
            if (
                !font?.previewSupported ||
                resolvedPreviewIds.current.has(faceId) ||
                failedPreviewIds.current.has(faceId) ||
                queuedPreviewIds.current.has(key) ||
                inFlightPreviewIds.current.has(key)
            ) {
                return;
            }
            queuedPreviewIds.current.add(key);
            pending.push({faceId, generation, sampleText: previewText});
        });

        if (pending.length === 0) {
            return;
        }

        setPreviewLoadingIds(current => {
            const next = {...current};
            pending.forEach(item => {
                next[item.faceId] = true;
            });
            return next;
        });

        if (priority) {
            previewQueue.current = [...pending, ...previewQueue.current];
        } else {
            previewQueue.current = [...previewQueue.current, ...pending];
        }
        drainPreviewQueue();
    }

    function drainPreviewQueue() {
        while (inFlightPreviewIds.current.size < PREVIEW_CONCURRENCY && previewQueue.current.length > 0) {
            const item = previewQueue.current.shift();
            if (!item) {
                continue;
            }

            const key = previewQueueKey(item.faceId, item.generation);
            queuedPreviewIds.current.delete(key);
            if (
                item.generation !== previewGeneration.current ||
                resolvedPreviewIds.current.has(item.faceId) ||
                failedPreviewIds.current.has(item.faceId) ||
                inFlightPreviewIds.current.has(key)
            ) {
                continue;
            }

            inFlightPreviewIds.current.add(key);
            loadPreviewFont(item)
                .finally(() => {
                    inFlightPreviewIds.current.delete(key);
                    drainPreviewQueue();
                });
        }
    }

    async function loadPreviewFont(request: PreviewQueueItem) {
        const {faceId, generation, sampleText: requestSampleText} = request;
        const font = fontById.current.get(faceId);
        const startedAt = performance.now();
        try {
            if (generation !== previewGeneration.current) {
                return;
            }

            const item = await api.getPreview(faceId, requestSampleText);
            if (generation !== previewGeneration.current) {
                return;
            }
            if (!item?.previewSupported || !item.fontUrl || !font) {
                failedPreviewIds.current.add(faceId);
                queuePreviewFailure(faceId);
                logFrontendWarning(`preview unavailable face=${faceId} message=${item?.message ?? 'empty response'}`);
                return;
            }

            const fontFace = new FontFace(item.fontFamily, `url("${item.fontUrl}")`, {
                style: font.italic ? 'italic' : 'normal',
                weight: String(font.weight || 400)
            });
            await fontFace.load();
            if (generation !== previewGeneration.current) {
                return;
            }
            document.fonts.add(fontFace);

            loadedFontFaces.current.set(faceId, fontFace);
            previewAccessTimes.current.set(faceId, Date.now());
            resolvedPreviewIds.current.add(faceId);
            queuePreviewLoaded(item);
            evictPreviewCache();

            const elapsed = Math.round(performance.now() - startedAt);
            const message = `preview loaded face=${faceId} cacheHit=${item.cacheHit} bytes=${item.byteSize} glyphs=${item.glyphCount ?? 0} missing=${item.missingRuneCount ?? 0} fallback=${Boolean(item.fallback)} reduction=${Math.round((item.reductionRatio ?? 0) * 100)}% elapsed=${elapsed}ms`;
            if (elapsed > PREVIEW_SLOW_MS) {
                logFrontendWarning(message);
            } else {
                logFrontendInfo(message);
            }
        } catch (error) {
            if (generation !== previewGeneration.current) {
                return;
            }
            failedPreviewIds.current.add(faceId);
            queuePreviewFailure(faceId);
            logFrontendWarning(`preview failed face=${faceId} error=${error instanceof Error ? error.message : String(error)}`);
        }
    }

    function queuePreviewLoaded(item: PreviewResponse) {
        pendingPreviewUpdates.current[item.faceId] = item;
        pendingSettledPreviewIds.current.add(item.faceId);
        schedulePreviewFlush();
    }

    function queuePreviewFailure(faceId: number) {
        pendingFailureUpdates.current[faceId] = true;
        pendingSettledPreviewIds.current.add(faceId);
        schedulePreviewFlush();
    }

    function schedulePreviewFlush() {
        if (previewFlushTimer.current !== null) {
            return;
        }
        previewFlushTimer.current = window.setTimeout(() => {
            previewFlushTimer.current = null;
            const loaded = pendingPreviewUpdates.current;
            const failed = pendingFailureUpdates.current;
            const settled = Array.from(pendingSettledPreviewIds.current);
            pendingPreviewUpdates.current = {};
            pendingFailureUpdates.current = {};
            pendingSettledPreviewIds.current.clear();

            if (Object.keys(loaded).length > 0) {
                setPreviews(current => ({...current, ...loaded}));
            }
            if (Object.keys(failed).length > 0) {
                setPreviewFailures(current => ({...current, ...failed}));
            }
            if (settled.length > 0) {
                setPreviewLoadingIds(current => {
                    const next = {...current};
                    settled.forEach(faceId => delete next[faceId]);
                    return next;
                });
            }
        }, 40);
    }

    function evictPreviewCache() {
        const evicted: number[] = [];
        while (loadedFontFaces.current.size > PREVIEW_LRU_LIMIT) {
            let evictFaceId: number | null = null;
            let oldest = Number.MAX_SAFE_INTEGER;
            loadedFontFaces.current.forEach((_, faceId) => {
                if (visiblePreviewIds.current.has(faceId) || selectedFaceRef.current === faceId) {
                    return;
                }
                const touched = previewAccessTimes.current.get(faceId) ?? 0;
                if (touched < oldest) {
                    oldest = touched;
                    evictFaceId = faceId;
                }
            });
            if (evictFaceId === null) {
                break;
            }

            const fontFace = loadedFontFaces.current.get(evictFaceId);
            if (fontFace) {
                document.fonts.delete(fontFace);
            }
            loadedFontFaces.current.delete(evictFaceId);
            previewAccessTimes.current.delete(evictFaceId);
            resolvedPreviewIds.current.delete(evictFaceId);
            evicted.push(evictFaceId);
        }

        if (evicted.length > 0) {
            setPreviews(current => {
                const next = {...current};
                evicted.forEach(faceId => delete next[faceId]);
                return next;
            });
            logFrontendInfo(`preview lru evicted=${evicted.length} loaded=${loadedFontFaces.current.size}`);
        }
    }

    function logFrontendInfo(message: string) {
        try {
            LogInfo(message);
        } catch {
            console.info(message);
        }
    }

    function logFrontendWarning(message: string) {
        try {
            LogWarning(message);
        } catch {
            console.warn(message);
        }
    }

    const shellStyle = {
        '--sidebar-width': `${layoutPrefs.sidebarWidth}px`,
        '--detail-width': `${layoutPrefs.detailWidth}px`
    } as CSSProperties;
    const currentLanguage: SupportedLanguage = i18n.language === 'en' ? 'en' : 'zh-CN';
    const effectiveAppInfo = appInfo ?? {
        name: t('app.name'),
        version: '0.2.0',
        dataDir: '',
        cacheDir: '',
        logDir: '',
        databasePath: ''
    };

    return (
        <div className="app-shell" style={shellStyle}>
            <aside className="sidebar">
                <div className="brand">
                    <div className="brand-subtitle">{t('app.subtitle')}</div>
                    <button className="settings-trigger" type="button" title={t('settings.open')} aria-label={t('settings.open')} onClick={() => {
                        closeActionMenu();
                        setSettingsOpen(true);
                    }}>
                        <Settings size={17}/>
                    </button>
                </div>

                <button className="primary-action" onClick={addRoot} disabled={busy}>
                    <FolderPlus size={18}/>
                    <span>{t('actions.addLibrary')}</span>
                </button>
                <div className="sidebar-action-row">
                    <button className="secondary-action" onClick={scanSystemFonts} disabled={busy}>
                        <Server size={16}/>
                        <span>{t('actions.scanSystemFonts')}</span>
                    </button>
                    <button className="secondary-action" onClick={rescanAllRoots} disabled={busy || roots.length === 0}>
                        <RefreshCw size={16}/>
                        <span>{t('actions.rescanAll')}</span>
                    </button>
                </div>

                <nav className="nav-list">
                    <button className={selectedRoot === 0 && !favoritesOnly && !installedOnly ? 'active' : ''} onClick={() => {
                        setSelectedRoot(0);
                        setSelectedFolder('');
                        setFavoritesOnly(false);
                        setInstalledOnly(false);
                    }}>
                        <Columns3 size={17}/>
                        <span>{t('nav.allIndexed')}</span>
                        <b>{fonts.length}</b>
                    </button>
                    <button className={favoritesOnly ? 'active' : ''} onClick={() => {
                        setFavoritesOnly(true);
                        setInstalledOnly(false);
                    }}>
                        <Star size={17}/>
                        <span>{t('nav.favorites')}</span>
                    </button>
                    <button className={installedOnly ? 'active' : ''} onClick={() => {
                        setInstalledOnly(true);
                        setFavoritesOnly(false);
                    }}>
                        <HardDriveDownload size={17}/>
                        <span>{t('nav.installed')}</span>
                    </button>
                </nav>

                <RootSection
                    title={t('nav.userLibraries')}
                    roots={userRoots}
                    busy={busy}
                    selectedRoot={selectedRoot}
                    selectedFolder={selectedFolder}
                    folders={folders}
                    expandedFolderKeys={expandedFolderKeys}
                    onSelectRoot={selectRoot}
                    onSelectFolder={selectFolder}
                    onToggleFolder={toggleFolder}
                    onOpenRootMenu={openRootActionMenu}
                    onOpenFolderMenu={openFolderActionMenu}
                    openActionMenuKey={actionMenu?.key ?? ''}
                    collapsedRootIds={collapsedRootIds}
                    t={t}
                />
                <RootSection
                    title={t('nav.systemLibraries')}
                    roots={systemRoots}
                    busy={busy}
                    selectedRoot={selectedRoot}
                    selectedFolder={selectedFolder}
                    folders={folders}
                    expandedFolderKeys={expandedFolderKeys}
                    onSelectRoot={selectRoot}
                    onSelectFolder={selectFolder}
                    onToggleFolder={toggleFolder}
                    onOpenRootMenu={openRootActionMenu}
                    onOpenFolderMenu={openFolderActionMenu}
                    openActionMenuKey={actionMenu?.key ?? ''}
                    collapsedRootIds={collapsedRootIds}
                    t={t}
                />
            </aside>

            <div
                className="resize-handle resize-handle-sidebar"
                role="separator"
                aria-label={t('accessibility.resizeSidebar')}
                aria-orientation="vertical"
                onPointerDown={(event) => startResizePane('sidebar', event)}
                onDoubleClick={() => resetPaneWidth('sidebar')}
            />

            <main className="workspace">
                <header className="toolbar">
                    <div className="search-box">
                        <Search size={18}/>
                        <input value={query} onChange={event => setQuery(event.target.value)} placeholder={t('toolbar.searchPlaceholder')} />
                    </div>
                    <div className="segmented view-segmented">
                        <button className={viewMode === 'list' ? 'active' : ''} onClick={activateListView} title={t('toolbar.list')}><List size={17}/></button>
                        <div
                            className={gridColumnMenuOpen ? 'grid-column-picker menu-open' : 'grid-column-picker'}
                            onPointerEnter={openGridColumnMenu}
                            onPointerLeave={closeGridColumnMenuAfterLeave}
                            onFocus={openGridColumnMenu}
                            onBlur={(event) => {
                                if (!event.currentTarget.contains(event.relatedTarget as Node | null)) {
                                    closeGridColumnMenuAfterLeave();
                                }
                            }}
                        >
                            <button
                                className={viewMode === 'grid' ? 'active' : ''}
                                onClick={activateGridView}
                                title={t('toolbar.grid')}
                                aria-haspopup="menu"
                                aria-expanded={gridColumnMenuOpen}
                            >
                                <Grid3X3 size={17}/>
                            </button>
                            {viewMode === 'grid' && gridColumnMenuOpen && (
                                <div className="grid-column-menu" role="menu" aria-label={t('toolbar.gridColumns')}>
                                    {CARD_COLUMN_OPTIONS.map(columns => (
                                        <button
                                            key={columns}
                                            role="menuitem"
                                            className={cardColumns === columns ? 'active' : ''}
                                            onClick={() => selectCardColumnCount(columns)}
                                            title={t('toolbar.columns', {count: columns})}
                                        >
                                            {t('toolbar.columns', {count: columns})}
                                        </button>
                                    ))}
                                </div>
                            )}
                        </div>
                    </div>
                </header>

                <div className="preview-controls">
                    <div className="scope-title">
                        <strong>{activeRoot?.name ?? (favoritesOnly ? t('nav.favorites') : installedOnly ? t('nav.installed') : t('nav.allFonts'))}</strong>
                        {selectedFolder && <span>{selectedFolder}</span>}
                    </div>
                    <input value={sampleText} onChange={event => setSampleText(event.target.value)} />
                    <label>
                        <span>{fontSize}px</span>
                        <input type="range" min="18" max="76" value={fontSize} onChange={event => setFontSize(Number(event.target.value))}/>
                    </label>
                </div>

                {activeRoot?.scanStatus === 'running' && (
                    <div className="scan-strip">
                        <span>{t('status.backgroundScanning', {processed: activeRoot.scanProcessed, total: activeRoot.scanTotal || '?'})}</span>
                        <div><i style={{width: `${scanPercent(activeRoot)}%`}} /></div>
                    </div>
                )}

                {notice && (
                    <div className={`notice ${notice.type}`}>
                        {notice.type === 'error' ? <AlertTriangle size={17}/> : notice.type === 'success' ? <CheckCircle2 size={17}/> : <Info size={17}/>}
                        <span>{notice.text}</span>
                        <button onClick={() => setNotice(null)} aria-label={t('accessibility.closeNotice')}><X size={15}/></button>
                    </div>
                )}

                <section
                    ref={resultsRef}
                    className={`font-results ${viewMode}`}
                    style={viewMode === 'grid' ? ({'--card-columns': String(cardColumns)} as CSSProperties) : undefined}
                    onScroll={handleResultsScroll}
                >
                    {fonts.map(font => (
                        <FontCard
                            key={font.faceId}
                            font={font}
                            preview={previews[font.faceId]}
                            previewLoading={Boolean(previewLoadingIds[font.faceId]) && !previewFailures[font.faceId]}
                            selected={selectedFace === font.faceId}
                            checked={checkedFaces.includes(font.faceId)}
                            sampleText={sampleText}
                            fontSize={fontSize}
                            onSelect={() => setSelectedFace(font.faceId)}
                            onCheck={() => toggleChecked(font.faceId)}
                            onFavorite={() => toggleFavorite(font)}
                            t={t}
                        />
                    ))}
                    {fonts.length === 0 && (
                        <div className="empty-state">
                            <FolderOpen size={36}/>
                            <strong>{t('empty.noFontsTitle')}</strong>
                            <span>{anyScanning ? t('empty.scanning') : t('empty.noResults')}</span>
                        </div>
                    )}
                    {hasMore && (
                        <div className="load-more-indicator">
                            {loadingMore ? t('empty.loadingMore') : t('empty.loadMore')}
                        </div>
                    )}
                </section>
            </main>

            <div
                className="resize-handle resize-handle-detail"
                role="separator"
                aria-label={t('accessibility.resizeDetail')}
                aria-orientation="vertical"
                onPointerDown={(event) => startResizePane('detail', event)}
                onDoubleClick={() => resetPaneWidth('detail')}
            />

            <aside className="detail-panel">
                <div className="detail-header">
                    <span>{t('detail.title')}</span>
                    <b>{activeIds.length > 0 ? t('status.selected', {count: activeIds.length}) : ''}</b>
                </div>
                {detail ? (
                    <>
                        <div className="detail-preview" style={{fontFamily: previews[detail.faceId]?.fontFamily, fontSize: Math.min(fontSize, 48)}}>
                            {sampleText}
                        </div>
                        <h2>{detail.fullName}</h2>
                        <div className="detail-tags">
                            <span>{detail.format}</span>
                            <span>{detail.style}</span>
                            <span>{detail.weight}</span>
                            {detail.isInstalled && <span className="installed">{t('status.installed')}</span>}
                        </div>
                        <div className="scope-toggle">
                            <button className={installScope === 'user' ? 'active' : ''} onClick={() => setInstallScope('user')}>{t('detail.currentUser')}</button>
                            <button className={installScope === 'machine' ? 'active' : ''} onClick={() => setInstallScope('machine')}>{t('detail.allUsers')}</button>
                        </div>
                        <div className="action-grid">
                            <button onClick={() => install('copy')} disabled={busy}><Download size={17}/>{t('actions.normalInstall')}</button>
                            <button onClick={() => install('link')} disabled={busy}><Link2 size={17}/>{t('actions.quickInstall')}</button>
                            <button onClick={uninstall} disabled={busy}><Trash2 size={17}/>{t('actions.uninstall')}</button>
                            <button onClick={() => api.revealInExplorer(detail.faceId).catch(showError)}><ExternalLink size={17}/>{t('actions.reveal')}</button>
                        </div>
                        <dl className="metadata">
                            <dt>{t('detail.family')}</dt><dd>{detail.family}</dd>
                            <dt>{t('detail.postScript')}</dt><dd>{detail.postScriptName || '-'}</dd>
                            <dt>{t('detail.file')}</dt><dd title={detail.path}>{detail.fileName}</dd>
                            <dt>{t('detail.size')}</dt><dd>{formatBytes(detail.size, t)}</dd>
                            <dt>{t('detail.glyphs')}</dt><dd>{detail.glyphCount || '-'}</dd>
                            <dt>{t('detail.version')}</dt><dd>{detail.version || '-'}</dd>
                            <dt>{t('detail.manufacturer')}</dt><dd>{detail.manufacturer || '-'}</dd>
                            <dt>{t('detail.status')}</dt><dd>{detail.error || detail.status}</dd>
                        </dl>
                        <div className="records">
                            <div className="records-title">{t('detail.installRecords')}</div>
                            {detail.installRecords?.slice(0, 5).map(record => (
                                <div className="record" key={record.id}>
                                    <span>{record.mode}/{record.scope}</span>
                                    <b>{record.status}</b>
                                </div>
                            ))}
                            {(!detail.installRecords || detail.installRecords.length === 0) && <span className="muted">{t('detail.noRecords')}</span>}
                        </div>
                    </>
                ) : (
                    <div className="empty-detail">
                        <Info size={30}/>
                        <span>{t('empty.noSelection')}</span>
                    </div>
                )}
            </aside>
            {renderActionMenu()}
            {settingsOpen && (
                <SettingsDialog
                    appInfo={effectiveAppInfo}
                    language={currentLanguage}
                    onLanguageChange={changeLanguage}
                    onClose={() => setSettingsOpen(false)}
                    t={t}
                />
            )}
            {operationProgress && <OperationProgressModal progress={operationProgress} t={t}/>}
        </div>
    )
}

function SettingsDialog(props: {
    appInfo: AppInfo;
    language: SupportedLanguage;
    onLanguageChange: (language: SupportedLanguage) => void;
    onClose: () => void;
    t: TFunction;
}) {
    const {appInfo, language, onLanguageChange, onClose, t} = props;

    useEffect(() => {
        const closeOnEscape = (event: KeyboardEvent) => {
            if (event.key === 'Escape') {
                onClose();
            }
        };
        window.addEventListener('keydown', closeOnEscape);
        return () => window.removeEventListener('keydown', closeOnEscape);
    }, [onClose]);

    return (
        <div className="settings-backdrop" role="presentation" onPointerDown={onClose}>
            <div className="settings-modal" role="dialog" aria-modal="true" aria-labelledby="settings-title" onPointerDown={event => event.stopPropagation()}>
                <button className="settings-close" type="button" onClick={onClose} aria-label={t('settings.close')} title={t('settings.close')}>
                    <X size={17}/>
                </button>
                <div className="settings-brand">
                    <img src={brandLogo} alt="" />
                    <div>
                        <h2 id="settings-title">{appInfo.name || t('app.name')}</h2>
                        <span>{t('settings.version')} {appInfo.version || '0.2.0'}</span>
                    </div>
                </div>
                <label className="settings-field">
                    <span>{t('settings.language')}</span>
                    <select value={language} onChange={event => onLanguageChange(event.target.value as SupportedLanguage)}>
                        {SUPPORTED_LANGUAGES.map(item => (
                            <option key={item} value={item}>
                                {item === 'zh-CN' ? t('settings.chinese') : t('settings.english')}
                            </option>
                        ))}
                    </select>
                </label>
            </div>
        </div>
    );
}

function OperationProgressModal(props: {progress: OperationProgress; t: TFunction}) {
    const {progress, t} = props;
    const total = Math.max(progress.total || 0, 0);
    const current = Math.min(progress.current || 0, total || progress.current || 0);
    const percent = total > 0 ? Math.max(4, Math.min(100, Math.round(current / total * 100))) : 12;
    const title = progress.operation === 'install' ? t('operation.installing') : t('operation.processingFonts');
    const status = progress.done ? t('operation.done') : progress.status === 'error' ? t('operation.partialFailed') : t('operation.processing');

    return (
        <div className="operation-backdrop" role="dialog" aria-modal="true" aria-label={title}>
            <div className="operation-modal">
                <div className="operation-title">
                    <strong>{title}</strong>
                    <span>{status}</span>
                </div>
                <div className="operation-progress">
                    <i style={{width: `${percent}%`}}/>
                </div>
                <div className="operation-stats">
                    <span>{current}/{total || '?'}</span>
                    <span>{t('operation.success', {count: progress.succeeded})}</span>
                    <span>{t('operation.failed', {count: progress.failed})}</span>
                </div>
                <div className="operation-file" title={progress.fileName || progress.message}>
                    {progress.fileName || progress.message || t('operation.preparing')}
                </div>
            </div>
        </div>
    );
}

function RootSection(props: {
    title: string;
    roots: LibraryRoot[];
    busy: boolean;
    selectedRoot: number;
    selectedFolder: string;
    folders: FontFolder[];
    expandedFolderKeys: string[];
    onSelectRoot: (id: number) => void;
    onSelectFolder: (rootId: number, path: string, hasChildren: boolean) => void;
    onToggleFolder: (rootId: number, path: string) => void;
    onOpenRootMenu: (root: LibraryRoot, event: ReactMouseEvent<HTMLButtonElement>) => void;
    onOpenFolderMenu: (root: LibraryRoot, folder: FontFolder, event: ReactMouseEvent<HTMLButtonElement>) => void;
    openActionMenuKey: string;
    collapsedRootIds: number[];
    t: TFunction;
}) {
    if (props.roots.length === 0) {
        return null;
    }
    return (
        <>
            <div className="section-label">{props.title}</div>
            <div className="root-list">
                {props.roots.map(root => {
                    const selected = props.selectedRoot === root.id;
                    const rootCollapsed = props.collapsedRootIds.includes(root.id);
                    const visibleFolders = selected && !rootCollapsed ? visibleFolderRows(root.id, props.folders, props.expandedFolderKeys) : [];
                    const scanning = root.scanStatus === 'running';
                    const rootKey = rootMenuKey(root);
                    return (
                        <div className="root-group" key={root.id}>
                            <div className={props.openActionMenuKey === rootKey ? 'root-row menu-open' : 'root-row'}>
                                <button
                                    className={selected && props.selectedFolder === '' ? 'root-main active' : 'root-main'}
                                    disabled={scanning}
                                    title={scanning ? props.t('status.scanning') : root.path}
                                    onClick={() => props.onSelectRoot(root.id)}
                                >
                                    {root.kind === 'system' ? <Server size={16}/> : <FolderOpen size={16}/>}
                                    <span>{root.name}</span>
                                    <b>{root.scanStatus === 'running' ? `${root.scanProcessed}/${root.scanTotal || '?'}` : root.fontCount}</b>
                                </button>
                                <button
                                    className="row-menu-trigger"
                                    title={props.t('menu.operations', {name: root.name})}
                                    aria-label={props.t('menu.operations', {name: root.name})}
                                    aria-haspopup="menu"
                                    aria-expanded={props.openActionMenuKey === rootKey}
                                    onClick={(event) => props.onOpenRootMenu(root, event)}
                                >
                                    <MoreHorizontal size={15}/>
                                </button>
                            </div>
                            {selected && visibleFolders.length > 0 && (
                                <div className="folder-children">
                                    {visibleFolders.map(row => {
                                        const folderKeyValue = folderMenuKey(root, row.folder);
                                        return (
                                        <div className={props.openActionMenuKey === folderKeyValue ? 'folder-row menu-open' : 'folder-row'} key={row.folder.path}>
                                            <button
                                                className="folder-toggle"
                                                disabled={scanning}
                                                aria-label={row.expanded ? props.t('folder.collapse') : props.t('folder.expand')}
                                                onClick={(event) => {
                                                    event.stopPropagation();
                                                    if (row.hasChildren) {
                                                        props.onToggleFolder(root.id, row.folder.path);
                                                    }
                                                }}
                                            >
                                                {row.hasChildren ? (row.expanded ? '-' : '+') : ''}
                                            </button>
                                            <button
                                                className={props.selectedFolder === row.folder.path ? 'folder-item active' : 'folder-item'}
                                                style={{paddingLeft: 4 + row.folder.depth * 8}}
                                                disabled={scanning}
                                                title={scanning ? props.t('status.scanning') : row.folder.path}
                                                onClick={() => props.onSelectFolder(root.id, row.folder.path, row.hasChildren)}
                                            >
                                                <FolderOpen size={14}/>
                                                <span>{row.folder.name}</span>
                                                <b>{row.folder.fontCount}</b>
                                            </button>
                                            <button
                                                className="row-menu-trigger"
                                                title={props.t('menu.operations', {name: row.folder.path})}
                                                aria-label={props.t('menu.operations', {name: row.folder.path})}
                                                aria-haspopup="menu"
                                                aria-expanded={props.openActionMenuKey === folderKeyValue}
                                                onClick={(event) => props.onOpenFolderMenu(root, row.folder, event)}
                                            >
                                                <MoreHorizontal size={14}/>
                                            </button>
                                        </div>
                                        );
                                    })}
                                </div>
                            )}
                        </div>
                    );
                })}
            </div>
        </>
    );
}

type FolderRow = {
    folder: FontFolder;
    hasChildren: boolean;
    expanded: boolean;
};

function visibleFolderRows(rootId: number, folders: FontFolder[], expandedKeys: string[]): FolderRow[] {
    const realFolders = folders.filter(folder => folder.path !== '');
    const children = new Map<string, FontFolder[]>();
    for (const folder of realFolders) {
        const parent = parentFolderPath(folder.path);
        const list = children.get(parent) ?? [];
        list.push(folder);
        children.set(parent, list);
    }
    for (const list of children.values()) {
        list.sort((a, b) => a.name.localeCompare(b.name, 'zh-Hans-CN'));
    }

    const rows: FolderRow[] = [];
    const appendChildren = (parent: string) => {
        for (const folder of children.get(parent) ?? []) {
            const hasChildren = (children.get(folder.path)?.length ?? 0) > 0;
            const expanded = expandedKeys.includes(folderKey(rootId, folder.path));
            rows.push({folder, hasChildren, expanded});
            if (hasChildren && expanded) {
                appendChildren(folder.path);
            }
        }
    };
    appendChildren('');
    return rows;
}

function parentFolderPath(path: string) {
    const index = path.lastIndexOf('/');
    if (index <= 0) {
        return '';
    }
    return path.slice(0, index);
}

function folderKey(rootId: number, path: string) {
    return `${rootId}:${path}`;
}

function rootMenuKey(root: LibraryRoot) {
    return `root:${root.id}`;
}

function folderMenuKey(root: LibraryRoot, folder: FontFolder) {
    return `folder:${root.id}:${folder.path}`;
}

function loadExpandedFolderKeys() {
    try {
        const raw = window.localStorage.getItem(EXPANDED_FOLDERS_KEY);
        const parsed = raw ? JSON.parse(raw) : [];
        return Array.isArray(parsed) ? parsed.filter(item => typeof item === 'string') : [];
    } catch {
        return [];
    }
}

function loadLayoutPrefs(): LayoutPrefs {
    try {
        const raw = window.localStorage.getItem(LAYOUT_PREFS_KEY);
        const parsed = raw ? JSON.parse(raw) : null;
        return clampLayoutPrefs({
            sidebarWidth: Number(parsed?.sidebarWidth) || DEFAULT_LAYOUT_PREFS.sidebarWidth,
            detailWidth: Number(parsed?.detailWidth) || DEFAULT_LAYOUT_PREFS.detailWidth
        });
    } catch {
        return DEFAULT_LAYOUT_PREFS;
    }
}

function persistLayoutPrefs(prefs: LayoutPrefs) {
    try {
        window.localStorage.setItem(LAYOUT_PREFS_KEY, JSON.stringify(clampLayoutPrefs(prefs)));
    } catch {
        // localStorage can be unavailable in restricted WebView contexts.
    }
}

function clampLayoutPrefs(prefs: LayoutPrefs): LayoutPrefs {
    const viewportWidth = typeof window === 'undefined' ? 1280 : window.innerWidth;
    const maxSidebarByViewport = viewportWidth - prefs.detailWidth - MIDDLE_MIN - RESIZE_HANDLE_TOTAL;
    const sidebarWidth = clamp(prefs.sidebarWidth, SIDEBAR_MIN, Math.max(SIDEBAR_MIN, Math.min(SIDEBAR_MAX, maxSidebarByViewport)));
    const maxDetailByViewport = viewportWidth - sidebarWidth - MIDDLE_MIN - RESIZE_HANDLE_TOTAL;
    const detailWidth = clamp(prefs.detailWidth, DETAIL_MIN, Math.max(DETAIL_MIN, Math.min(DETAIL_MAX, maxDetailByViewport)));
    return {sidebarWidth, detailWidth};
}

function clamp(value: number, min: number, max: number) {
    return Math.max(min, Math.min(max, value));
}

function menuPosition(rect: DOMRect, estimatedHeight: number) {
    const width = 188;
    const left = clamp(rect.right - width, 8, window.innerWidth - width - 8);
    const top = clamp(rect.bottom + 4, 8, window.innerHeight - estimatedHeight - 8);
    return {left, top};
}

const FontCard = memo(function FontCard(props: {
    font: FontItem;
    preview?: PreviewResponse;
    previewLoading: boolean;
    selected: boolean;
    checked: boolean;
    sampleText: string;
    fontSize: number;
    onSelect: () => void;
    onCheck: () => void;
    onFavorite: () => void;
    t: TFunction;
}) {
    const {font, preview, previewLoading, selected, checked, sampleText, fontSize, onSelect, onCheck, onFavorite, t} = props;
    const fontFamily = preview?.previewSupported ? preview.fontFamily : undefined;
    const text = sampleText || preview?.sampleText || font.fullName;

    return (
        <article
            className={`font-card ${selected ? 'selected' : ''} ${font.status === 'limited' ? 'limited' : ''}`}
            data-face-id={font.faceId}
            onClick={onSelect}
        >
            <div className="card-topline">
                <label className="check" onClick={event => event.stopPropagation()}>
                    <input type="checkbox" checked={checked} onChange={onCheck}/>
                </label>
                <div className="card-title">
                    <span className="font-family-name">{font.family}</span>
                    <span className="font-file-name" title={font.fileName}>{font.fileName}</span>
                </div>
                <button className={font.isFavorite ? 'star active' : 'star'} onClick={event => {
                    event.stopPropagation();
                    onFavorite();
                }} title={t('accessibility.favorite')} aria-label={t('accessibility.favorite')}>
                    <Star size={16}/>
                </button>
            </div>
            <div
                className={previewLoading ? 'sample-line preview-loading' : 'sample-line'}
                style={{fontFamily, fontSize, fontStyle: font.italic ? 'italic' : 'normal', fontWeight: font.weight || 400}}
            >
                {font.previewSupported ? text : font.fullName}
            </div>
            <div className="card-meta">
                <span>{font.style}</span>
                <span>{font.format}</span>
                {font.isInstalled && <span className="pill">{t('status.installed')}</span>}
                {font.status !== 'ok' && <span className="warning">{t('status.limited')}</span>}
            </div>
        </article>
    );
});

function scanPercent(root: LibraryRoot) {
    if (!root.scanTotal) {
        return 3;
    }
    return Math.max(3, Math.min(100, Math.round(root.scanProcessed / root.scanTotal * 100)));
}

function formatBytes(value: number, t: TFunction) {
    if (!value) {
        return '-';
    }
    if (value < 1024) {
        return t('units.bytes', {value});
    }
    if (value < 1024 * 1024) {
        return t('units.kilobytes', {value: (value / 1024).toFixed(1)});
    }
    return t('units.megabytes', {value: (value / 1024 / 1024).toFixed(1)});
}

export default App
