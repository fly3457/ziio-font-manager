import i18n from 'i18next';
import {initReactI18next} from 'react-i18next';

export const LANGUAGE_STORAGE_KEY = 'ziio.fontManager.language.v1';
export const SUPPORTED_LANGUAGES = ['zh-CN', 'en'] as const;
export type SupportedLanguage = typeof SUPPORTED_LANGUAGES[number];
export const DEFAULT_SAMPLE_TEXTS = {
    'zh-CN': '永字八法 AaBbCc 0123456789',
    en: 'AaBbCc 0123456789 The quick brown fox'
} satisfies Record<SupportedLanguage, string>;

const resources = {
    'zh-CN': {
        translation: {
            app: {
                name: 'Ziio Font Manager',
                subtitle: '本地字体库管理'
            },
            settings: {
                title: '设置',
                close: '关闭设置',
                language: '语言',
                version: '版本',
                chinese: '中文',
                english: 'English',
                open: '打开设置'
            },
            actions: {
                addLibrary: '添加字体库',
                scanSystemFonts: '扫描系统字库',
                rescanAll: '全量重新扫描',
                syncLibrary: '同步字体库',
                removeLibrary: '移除字体库',
                syncFolder: '同步文件夹',
                normalInstall: '普通安装',
                quickInstall: '快捷安装',
                uninstall: '卸载',
                reveal: '位置'
            },
            nav: {
                allIndexed: '全部已索引字体',
                favorites: '收藏',
                installed: '已安装',
                userLibraries: '用户字体库',
                systemLibraries: '系统字库',
                allFonts: '全部字体'
            },
            toolbar: {
                searchPlaceholder: '搜索 family、样式、文件名',
                list: '列表',
                grid: '网格',
                gridColumns: '网格列数',
                columns: '{{count}}列'
            },
            preview: {
                sample: DEFAULT_SAMPLE_TEXTS['zh-CN']
            },
            status: {
                scanning: '扫描中',
                backgroundScanning: '正在后台扫描：{{processed}}/{{total}}',
                selected: '{{count}} 选中',
                installed: '已安装',
                limited: '受限'
            },
            menu: {
                operations: '{{name}} 操作',
                scanningHint: '扫描中'
            },
            folder: {
                expand: '展开文件夹',
                collapse: '收起文件夹'
            },
            empty: {
                noFontsTitle: '没有字体结果',
                scanning: '后台扫描中，索引到字体后会逐步显示。',
                noResults: '当前筛选条件下没有可显示的字体。',
                loadingMore: '加载中...',
                loadMore: '继续向下滚动加载更多',
                noSelection: '未选择字体'
            },
            detail: {
                title: '属性',
                currentUser: '当前用户',
                allUsers: '所有用户',
                family: 'Family',
                postScript: 'PostScript',
                file: '文件',
                size: '大小',
                glyphs: '字形',
                version: '版本',
                manufacturer: '厂商',
                status: '状态',
                installRecords: '安装记录',
                noRecords: '无记录'
            },
            operation: {
                installing: '正在安装字体',
                processingFonts: '正在处理字体',
                done: '处理完成',
                partialFailed: '部分失败',
                processing: '处理中',
                success: '成功 {{count}}',
                failed: '失败 {{count}}',
                preparing: '准备中...'
            },
            notices: {
                addedRoot: '已添加 {{name}}，正在后台扫描',
                scanSystemStarted: '已开始扫描 {{count}} 个系统字体目录',
                noRoots: '尚未添加字体库',
                rescanAllStarted: '已开始全量重新扫描：{{count}} 个字体库',
                allRootsScanning: '所有字体库都在扫描中',
                rootScanningSync: '字体库正在扫描，请等待完成后再同步',
                rootScanningFolderSync: '字体库正在扫描，请等待完成后再同步文件夹',
                syncRootStarted: '已开始同步字体库：{{name}}',
                syncFolderStarted: '已开始同步文件夹：{{path}}',
                systemRootCannotDelete: '系统字库不能删除',
                rootScanningDelete: '字体库正在扫描，请等待扫描完成后再删除',
                removedRoot: '已从 Ziio 移除 {{name}}，源文件未删除',
                chooseFont: '请选择字体',
                operationFailed: '{{succeeded}} 成功，{{failed}} 失败{{message}}',
                operationCompleted: '{{count}} 项完成',
                errorPrefix: '：{{message}}'
            },
            confirm: {
                removeRoot: '从 Ziio 移除字体库“{{name}}”？\n\n路径：{{path}}\n\n这只会删除应用内索引记录，磁盘上的文件夹和字体文件会保留。',
                uninstall: '卸载 {{count}} 个字体记录？'
            },
            accessibility: {
                resizeSidebar: '调整左侧栏宽度',
                resizeDetail: '调整右侧属性栏宽度',
                favorite: '收藏',
                closeNotice: '关闭提示'
            },
            units: {
                bytes: '{{value}} B',
                kilobytes: '{{value}} KB',
                megabytes: '{{value}} MB'
            }
        }
    },
    en: {
        translation: {
            app: {
                name: 'Ziio Font Manager',
                subtitle: 'Local font library manager'
            },
            settings: {
                title: 'Settings',
                close: 'Close settings',
                language: 'Language',
                version: 'Version',
                chinese: '中文',
                english: 'English',
                open: 'Open settings'
            },
            actions: {
                addLibrary: 'Add Library',
                scanSystemFonts: 'Scan System',
                rescanAll: 'Rescan All',
                syncLibrary: 'Sync Library',
                removeLibrary: 'Remove Library',
                syncFolder: 'Sync Folder',
                normalInstall: 'Install',
                quickInstall: 'Quick Install',
                uninstall: 'Uninstall',
                reveal: 'Location'
            },
            nav: {
                allIndexed: 'All Indexed Fonts',
                favorites: 'Favorites',
                installed: 'Installed',
                userLibraries: 'User Libraries',
                systemLibraries: 'System Libraries',
                allFonts: 'All Fonts'
            },
            toolbar: {
                searchPlaceholder: 'Search family, style, file name',
                list: 'List',
                grid: 'Grid',
                gridColumns: 'Grid columns',
                columns: '{{count}} columns'
            },
            preview: {
                sample: DEFAULT_SAMPLE_TEXTS.en
            },
            status: {
                scanning: 'Scanning',
                backgroundScanning: 'Scanning in background: {{processed}}/{{total}}',
                selected: '{{count}} selected',
                installed: 'Installed',
                limited: 'Limited'
            },
            menu: {
                operations: '{{name}} actions',
                scanningHint: 'Scanning'
            },
            folder: {
                expand: 'Expand folder',
                collapse: 'Collapse folder'
            },
            empty: {
                noFontsTitle: 'No Font Results',
                scanning: 'Background scan is running. Fonts will appear as they are indexed.',
                noResults: 'No fonts match the current filters.',
                loadingMore: 'Loading...',
                loadMore: 'Scroll down to load more',
                noSelection: 'No font selected'
            },
            detail: {
                title: 'Properties',
                currentUser: 'Current User',
                allUsers: 'All Users',
                family: 'Family',
                postScript: 'PostScript',
                file: 'File',
                size: 'Size',
                glyphs: 'Glyphs',
                version: 'Version',
                manufacturer: 'Manufacturer',
                status: 'Status',
                installRecords: 'Install Records',
                noRecords: 'No records'
            },
            operation: {
                installing: 'Installing Fonts',
                processingFonts: 'Processing Fonts',
                done: 'Done',
                partialFailed: 'Partially Failed',
                processing: 'Processing',
                success: '{{count}} succeeded',
                failed: '{{count}} failed',
                preparing: 'Preparing...'
            },
            notices: {
                addedRoot: 'Added {{name}}. Background scan started.',
                scanSystemStarted: 'Started scanning {{count}} system font folders',
                noRoots: 'No font libraries added yet',
                rescanAllStarted: 'Started full rescan for {{count}} font libraries',
                allRootsScanning: 'All font libraries are already scanning',
                rootScanningSync: 'This font library is scanning. Please sync after it finishes.',
                rootScanningFolderSync: 'This font library is scanning. Please sync folders after it finishes.',
                syncRootStarted: 'Started syncing library: {{name}}',
                syncFolderStarted: 'Started syncing folder: {{path}}',
                systemRootCannotDelete: 'System font libraries cannot be removed',
                rootScanningDelete: 'This font library is scanning. Please remove it after the scan finishes.',
                removedRoot: 'Removed {{name}} from Ziio. Source files were not deleted.',
                chooseFont: 'Select a font first',
                operationFailed: '{{succeeded}} succeeded, {{failed}} failed{{message}}',
                operationCompleted: '{{count}} items completed',
                errorPrefix: ': {{message}}'
            },
            confirm: {
                removeRoot: 'Remove “{{name}}” from Ziio?\n\nPath: {{path}}\n\nThis only removes the app index. The source folder and font files on disk will remain.',
                uninstall: 'Uninstall {{count}} font records?'
            },
            accessibility: {
                resizeSidebar: 'Resize sidebar',
                resizeDetail: 'Resize details panel',
                favorite: 'Favorite',
                closeNotice: 'Close notice'
            },
            units: {
                bytes: '{{value}} B',
                kilobytes: '{{value}} KB',
                megabytes: '{{value}} MB'
            }
        }
    }
};

function initialLanguage(): SupportedLanguage {
    try {
        const saved = window.localStorage.getItem(LANGUAGE_STORAGE_KEY);
        if (saved === 'zh-CN' || saved === 'en') {
            return saved;
        }
    } catch {
        // localStorage can be unavailable in restricted WebView contexts.
    }
    return 'zh-CN';
}

i18n.use(initReactI18next).init({
    resources,
    lng: initialLanguage(),
    fallbackLng: 'zh-CN',
    interpolation: {
        escapeValue: false
    }
});

export default i18n;
