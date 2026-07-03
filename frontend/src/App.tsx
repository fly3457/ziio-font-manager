import {memo, type CSSProperties, type UIEvent, useEffect, useRef, useState} from 'react';
import './App.css';
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
    RefreshCw,
    Search,
    Server,
    Star,
    Trash2,
    X
} from 'lucide-react';
import {EventsOn, LogInfo, LogWarning} from '../wailsjs/runtime/runtime';
import {api, FontDetail, FontFolder, FontItem, LibraryRoot, OperationProgress, OperationResult, PreviewResponse} from './api';

const PAGE_SIZE = 20;
const PREVIEW_CONCURRENCY = 2;
const PREVIEW_LRU_LIMIT = 1000;
const PREVIEW_ROOT_MARGIN = '900px 0px';
const PREVIEW_SLOW_MS = 300;
const EXPANDED_FOLDERS_KEY = 'ziio.fontManager.expandedFolders.v1';
const CARD_COLUMN_OPTIONS = [2, 3, 4, 5] as const;

type ViewMode = 'list' | 'grid';
type CardColumnCount = typeof CARD_COLUMN_OPTIONS[number];
type PreviewQueueItem = {faceId: number; generation: number; sampleText: string};

function App() {
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
    const [sampleText, setSampleText] = useState('永字八法 AaBbCc 0123456789');
    const [fontSize, setFontSize] = useState(34);
    const [busy, setBusy] = useState(false);
    const [loadingMore, setLoadingMore] = useState(false);
    const [hasMore, setHasMore] = useState(false);
    const [notice, setNotice] = useState<{type: 'info' | 'error' | 'success', text: string} | null>(null);
    const [operationProgress, setOperationProgress] = useState<OperationProgress | null>(null);

    const activeIds = checkedFaces.length > 0 ? checkedFaces : selectedFace ? [selectedFace] : [];
    const activeRoot = roots.find(root => root.id === selectedRoot);
    const userRoots = roots.filter(root => root.kind !== 'system');
    const systemRoots = roots.filter(root => root.kind === 'system');
    const anyScanning = roots.some(root => root.scanStatus === 'running');
    const resultsRef = useRef<HTMLElement | null>(null);
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
    }, []);

    useEffect(() => {
        window.localStorage.setItem(EXPANDED_FOLDERS_KEY, JSON.stringify(expandedFolderKeys));
    }, [expandedFolderKeys]);

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

    async function loadFonts(reset: boolean) {
        const offset = reset ? 0 : fonts.length;
        try {
            const next = await api.searchFonts({
                query,
                rootId: selectedRoot,
                folderPath: selectedFolder,
                folderRecursive: true,
                favoritesOnly,
                installedOnly,
                limit: PAGE_SIZE,
                offset
            });
            setHasMore((next?.length ?? 0) === PAGE_SIZE);
            if (reset) {
                setFonts(next ?? []);
                setCheckedFaces([]);
                setSelectedFace(next?.length ? next[0].faceId : null);
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
            setNotice({type: 'success', text: `已添加 ${root.name}，正在后台扫描`});
            await loadRoots();
            await loadFolders(root.id);
            await loadFonts(true);
        });
    }

    async function scanSystemFonts() {
        await runBusy(async () => {
            const next = await api.scanSystemFonts();
            setNotice({type: 'success', text: `已开始扫描 ${next.length} 个系统字体目录`});
            await loadRoots();
            if (next[0]) {
                setSelectedRoot(next[0].id);
                setSelectedFolder('');
            }
        });
    }

    async function rescanRoot() {
        const target = selectedRoot || roots[0]?.id;
        if (!target) {
            setNotice({type: 'info', text: '尚未添加字体库'});
            return;
        }
        await runBusy(async () => {
            await api.rescanRoot(target);
            setNotice({type: 'success', text: '已开始后台扫描'});
            await loadRoots();
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
            setNotice({type: 'info', text: '请选择字体'});
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
            message: 'preparing install',
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
            setNotice({type: 'info', text: '请选择字体'});
            return;
        }
        if (!window.confirm(`卸载 ${activeIds.length} 个字体记录？`)) {
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
        setSelectedRoot(rootId);
        setSelectedFolder('');
        setFavoritesOnly(false);
        setInstalledOnly(false);
    }

    function selectFolder(rootId: number, path: string, hasChildren: boolean) {
        setSelectedFolder(path);
        if (hasChildren) {
            expandFolder(rootId, path);
        }
    }

    function expandFolder(rootId: number, path: string) {
        const key = folderKey(rootId, path);
        setExpandedFolderKeys(current => current.includes(key) ? current : [...current, key]);
    }

    function toggleFolder(rootId: number, path: string) {
        const key = folderKey(rootId, path);
        setExpandedFolderKeys(current => current.includes(key) ? current.filter(item => item !== key) : [...current, key]);
    }

    function applyOperationNotice(result: OperationResult) {
        const firstError = result.messages?.find(m => m.level === 'error')?.message;
        if ((result.failed ?? 0) > 0) {
            setNotice({type: 'error', text: `${result.succeeded} 成功，${result.failed} 失败${firstError ? `：${firstError}` : ''}`});
        } else {
            setNotice({type: 'success', text: `${result.succeeded} 项完成`});
        }
    }

    function showError(error: unknown) {
        const text = error instanceof Error ? error.message : String(error);
        setNotice({type: 'error', text});
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

    return (
        <div className="app-shell">
            <aside className="sidebar">
                <div className="brand">
                    <div className="brand-mark">Z</div>
                    <div>
                        <div className="brand-title">Ziio Font Manager</div>
                        <div className="brand-subtitle">本地字体库管理</div>
                    </div>
                </div>

                <button className="primary-action" onClick={addRoot} disabled={busy}>
                    <FolderPlus size={18}/>
                    <span>添加字体库</span>
                </button>
                <button className="secondary-action" onClick={scanSystemFonts} disabled={busy}>
                    <Server size={17}/>
                    <span>扫描系统字库</span>
                </button>

                <nav className="nav-list">
                    <button className={selectedRoot === 0 && !favoritesOnly && !installedOnly ? 'active' : ''} onClick={() => {
                        setSelectedRoot(0);
                        setSelectedFolder('');
                        setFavoritesOnly(false);
                        setInstalledOnly(false);
                    }}>
                        <Columns3 size={17}/>
                        <span>全部已索引字体</span>
                        <b>{fonts.length}</b>
                    </button>
                    <button className={favoritesOnly ? 'active' : ''} onClick={() => {
                        setFavoritesOnly(true);
                        setInstalledOnly(false);
                    }}>
                        <Star size={17}/>
                        <span>收藏</span>
                    </button>
                    <button className={installedOnly ? 'active' : ''} onClick={() => {
                        setInstalledOnly(true);
                        setFavoritesOnly(false);
                    }}>
                        <HardDriveDownload size={17}/>
                        <span>已安装</span>
                    </button>
                </nav>

                <RootSection
                    title="用户字体库"
                    roots={userRoots}
                    selectedRoot={selectedRoot}
                    selectedFolder={selectedFolder}
                    folders={folders}
                    expandedFolderKeys={expandedFolderKeys}
                    onSelectRoot={selectRoot}
                    onSelectFolder={selectFolder}
                    onToggleFolder={toggleFolder}
                />
                <RootSection
                    title="系统字库"
                    roots={systemRoots}
                    selectedRoot={selectedRoot}
                    selectedFolder={selectedFolder}
                    folders={folders}
                    expandedFolderKeys={expandedFolderKeys}
                    onSelectRoot={selectRoot}
                    onSelectFolder={selectFolder}
                    onToggleFolder={toggleFolder}
                />
            </aside>

            <main className="workspace">
                <header className="toolbar">
                    <div className="search-box">
                        <Search size={18}/>
                        <input value={query} onChange={event => setQuery(event.target.value)} placeholder="搜索 family、样式、文件名" />
                    </div>
                    <button className="icon-button" onClick={rescanRoot} disabled={busy} title="重新扫描">
                        <RefreshCw size={18}/>
                    </button>
                    <div className="segmented">
                        <button className={viewMode === 'list' ? 'active' : ''} onClick={() => setViewMode('list')} title="列表"><List size={17}/></button>
                        <button className={viewMode === 'grid' ? 'active' : ''} onClick={() => setViewMode('grid')} title="网格"><Grid3X3 size={17}/></button>
                    </div>
                    {viewMode === 'grid' && (
                        <div className="segmented column-segmented" aria-label="卡片列数">
                            {CARD_COLUMN_OPTIONS.map(columns => (
                                <button
                                    key={columns}
                                    className={cardColumns === columns ? 'active' : ''}
                                    onClick={() => setCardColumns(columns)}
                                    title={`${columns}列`}
                                >
                                    {columns}
                                </button>
                            ))}
                        </div>
                    )}
                </header>

                <div className="preview-controls">
                    <div className="scope-title">
                        <strong>{activeRoot?.name ?? (favoritesOnly ? '收藏' : installedOnly ? '已安装' : '全部字体')}</strong>
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
                        <span>正在后台扫描：{activeRoot.scanProcessed}/{activeRoot.scanTotal || '?'}</span>
                        <div><i style={{width: `${scanPercent(activeRoot)}%`}} /></div>
                    </div>
                )}

                {notice && (
                    <div className={`notice ${notice.type}`}>
                        {notice.type === 'error' ? <AlertTriangle size={17}/> : notice.type === 'success' ? <CheckCircle2 size={17}/> : <Info size={17}/>}
                        <span>{notice.text}</span>
                        <button onClick={() => setNotice(null)}><X size={15}/></button>
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
                        />
                    ))}
                    {fonts.length === 0 && (
                        <div className="empty-state">
                            <FolderOpen size={36}/>
                            <strong>没有字体结果</strong>
                            <span>{anyScanning ? '后台扫描中，索引到字体后会逐步显示。' : '当前筛选条件下没有可显示的字体。'}</span>
                        </div>
                    )}
                    {hasMore && (
                        <div className="load-more-indicator">
                            {loadingMore ? '加载中...' : '继续向下滚动加载更多'}
                        </div>
                    )}
                </section>
            </main>

            <aside className="detail-panel">
                <div className="detail-header">
                    <span>属性</span>
                    <b>{activeIds.length > 0 ? `${activeIds.length} 选中` : ''}</b>
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
                            {detail.isInstalled && <span className="installed">已安装</span>}
                        </div>
                        <div className="scope-toggle">
                            <button className={installScope === 'user' ? 'active' : ''} onClick={() => setInstallScope('user')}>当前用户</button>
                            <button className={installScope === 'machine' ? 'active' : ''} onClick={() => setInstallScope('machine')}>所有用户</button>
                        </div>
                        <div className="action-grid">
                            <button onClick={() => install('copy')} disabled={busy}><Download size={17}/>普通安装</button>
                            <button onClick={() => install('link')} disabled={busy}><Link2 size={17}/>快捷安装</button>
                            <button onClick={uninstall} disabled={busy}><Trash2 size={17}/>卸载</button>
                            <button onClick={() => api.revealInExplorer(detail.faceId).catch(showError)}><ExternalLink size={17}/>位置</button>
                        </div>
                        <dl className="metadata">
                            <dt>Family</dt><dd>{detail.family}</dd>
                            <dt>PostScript</dt><dd>{detail.postScriptName || '-'}</dd>
                            <dt>文件</dt><dd title={detail.path}>{detail.fileName}</dd>
                            <dt>大小</dt><dd>{formatBytes(detail.size)}</dd>
                            <dt>字形</dt><dd>{detail.glyphCount || '-'}</dd>
                            <dt>版本</dt><dd>{detail.version || '-'}</dd>
                            <dt>厂商</dt><dd>{detail.manufacturer || '-'}</dd>
                            <dt>状态</dt><dd>{detail.error || detail.status}</dd>
                        </dl>
                        <div className="records">
                            <div className="records-title">安装记录</div>
                            {detail.installRecords?.slice(0, 5).map(record => (
                                <div className="record" key={record.id}>
                                    <span>{record.mode}/{record.scope}</span>
                                    <b>{record.status}</b>
                                </div>
                            ))}
                            {(!detail.installRecords || detail.installRecords.length === 0) && <span className="muted">无记录</span>}
                        </div>
                    </>
                ) : (
                    <div className="empty-detail">
                        <Info size={30}/>
                        <span>未选择字体</span>
                    </div>
                )}
            </aside>
            {operationProgress && <OperationProgressModal progress={operationProgress}/>}
        </div>
    )
}

function OperationProgressModal(props: {progress: OperationProgress}) {
    const {progress} = props;
    const total = Math.max(progress.total || 0, 0);
    const current = Math.min(progress.current || 0, total || progress.current || 0);
    const percent = total > 0 ? Math.max(4, Math.min(100, Math.round(current / total * 100))) : 12;
    const title = progress.operation === 'install' ? '正在安装字体' : '正在处理字体';
    const status = progress.done ? '处理完成' : progress.status === 'error' ? '部分失败' : '处理中';

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
                    <span>成功 {progress.succeeded}</span>
                    <span>失败 {progress.failed}</span>
                </div>
                <div className="operation-file" title={progress.fileName || progress.message}>
                    {progress.fileName || progress.message || '准备中...'}
                </div>
            </div>
        </div>
    );
}

function RootSection(props: {
    title: string;
    roots: LibraryRoot[];
    selectedRoot: number;
    selectedFolder: string;
    folders: FontFolder[];
    expandedFolderKeys: string[];
    onSelectRoot: (id: number) => void;
    onSelectFolder: (rootId: number, path: string, hasChildren: boolean) => void;
    onToggleFolder: (rootId: number, path: string) => void;
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
                    const visibleFolders = selected ? visibleFolderRows(root.id, props.folders, props.expandedFolderKeys) : [];
                    return (
                        <div className="root-group" key={root.id}>
                            <button className={selected && props.selectedFolder === '' ? 'active' : ''} onClick={() => props.onSelectRoot(root.id)}>
                                {root.kind === 'system' ? <Server size={16}/> : <FolderOpen size={16}/>}
                                <span>{root.name}</span>
                                <b>{root.scanStatus === 'running' ? `${root.scanProcessed}/${root.scanTotal || '?'}` : root.fontCount}</b>
                            </button>
                            {selected && visibleFolders.length > 0 && (
                                <div className="folder-children">
                                    {visibleFolders.map(row => (
                                        <div className="folder-row" key={row.folder.path}>
                                            <button
                                                className="folder-toggle"
                                                aria-label={row.expanded ? '收起文件夹' : '展开文件夹'}
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
                                                style={{paddingLeft: 8 + row.folder.depth * 12}}
                                                onClick={() => props.onSelectFolder(root.id, row.folder.path, row.hasChildren)}
                                            >
                                                <FolderOpen size={14}/>
                                                <span>{row.folder.name}</span>
                                                <b>{row.folder.fontCount}</b>
                                            </button>
                                        </div>
                                    ))}
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

function loadExpandedFolderKeys() {
    try {
        const raw = window.localStorage.getItem(EXPANDED_FOLDERS_KEY);
        const parsed = raw ? JSON.parse(raw) : [];
        return Array.isArray(parsed) ? parsed.filter(item => typeof item === 'string') : [];
    } catch {
        return [];
    }
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
}) {
    const {font, preview, previewLoading, selected, checked, sampleText, fontSize, onSelect, onCheck, onFavorite} = props;
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
                }} title="收藏">
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
                {font.isInstalled && <span className="pill">已安装</span>}
                {font.status !== 'ok' && <span className="warning">受限</span>}
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

function formatBytes(value: number) {
    if (!value) {
        return '-';
    }
    if (value < 1024) {
        return `${value} B`;
    }
    if (value < 1024 * 1024) {
        return `${(value / 1024).toFixed(1)} KB`;
    }
    return `${(value / 1024 / 1024).toFixed(1)} MB`;
}

export default App
