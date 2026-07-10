export type AppInfo = {
  name: string;
  version: string;
  dataDir: string;
  cacheDir: string;
  logDir: string;
  databasePath: string;
};

export type LibraryRoot = {
  id: number;
  path: string;
  name: string;
  kind: "user" | "system" | string;
  enabled: boolean;
  createdAt: string;
  updatedAt: string;
  lastScanAt: string;
  fontCount: number;
  scanStatus: string;
  scanTotal: number;
  scanProcessed: number;
};

export type FontFolder = {
  rootId: number;
  path: string;
  name: string;
  depth: number;
  fontCount: number;
};

export type ScanResult = {
  rootId: number;
  total: number;
  processed: number;
  added: number;
  updated: number;
  failed: number;
  missing: number;
  unchanged: number;
  scope: string;
  scopePath: string;
};

export type FontItem = {
  faceId: number;
  fileId: number;
  rootId: number;
  rootPath: string;
  path: string;
  fileName: string;
  format: string;
  family: string;
  style: string;
  fullName: string;
  postScriptName: string;
  weight: number;
  italic: boolean;
  isFavorite: boolean;
  isInstalled: boolean;
  previewSupported: boolean;
  status: string;
  error: string;
  updatedAt: string;
};

export type InstallRecord = {
  id: number;
  fileId: number;
  faceId: number;
  sourcePath: string;
  targetPath: string;
  mode: string;
  scope: string;
  registryKey: string;
  registryValueName: string;
  registryValueData: string;
  installedAt: string;
  uninstalledAt: string;
  status: string;
  error: string;
};

export type FontDetail = FontItem & {
  size: number;
  modifiedAt: string;
  hash: string;
  sampleText: string;
  manufacturer: string;
  designer: string;
  license: string;
  version: string;
  glyphCount: number;
  installRecords: InstallRecord[];
};

export type PreviewResponse = {
  faceId: number;
  fontFamily: string;
  css: string;
  fontUrl: string;
  sampleText: string;
  previewSupported: boolean;
  message: string;
  cacheHit: boolean;
  byteSize: number;
  glyphCount: number;
  missingRuneCount: number;
  fullBytes: number;
  subsetBytes: number;
  fallback: boolean;
  fallbackReason: string;
  reductionRatio: number;
};

export type OperationResult = {
  operation: string;
  succeeded: number;
  failed: number;
  messages: { faceId: number; fileId: number; level: string; message: string }[];
};

export type OperationProgress = {
  operation: string;
  mode: string;
  scope: string;
  current: number;
  total: number;
  succeeded: number;
  failed: number;
  faceId: number;
  fileId: number;
  fileName: string;
  status: string;
  message: string;
  done: boolean;
};

export type FontQuery = {
  query: string;
  rootId: number;
  folderPath: string;
  folderRecursive: boolean;
  favoritesOnly: boolean;
  installedOnly: boolean;
  limit: number;
  offset: number;
};

type WailsWindow = Window & {
  go?: {
    library?: {
      LibraryService?: Record<string, (...args: any[]) => Promise<any>>;
      FontService?: Record<string, (...args: any[]) => Promise<any>>;
      InstallService?: Record<string, (...args: any[]) => Promise<any>>;
    };
    main?: {
      App?: Record<string, (...args: any[]) => Promise<any>>;
    };
  };
};

function service(name: "LibraryService" | "FontService" | "InstallService") {
  const w = window as WailsWindow;
  const svc = w.go?.library?.[name];
  if (!svc) {
    throw new Error("Wails bridge is not ready. Start the app with Wails to use native font management.");
  }
  return svc;
}

export const api = {
  appInfo: () => {
    const w = window as WailsWindow;
    return (w.go?.main?.App?.GetAppInfo?.() as Promise<AppInfo>) ?? Promise.resolve(null);
  },
  chooseAndAddRoot: () => service("LibraryService").ChooseAndAddRoot() as Promise<LibraryRoot>,
  scanSystemFonts: () => service("LibraryService").ScanSystemFonts() as Promise<LibraryRoot[]>,
  listRoots: () => service("LibraryService").ListRoots() as Promise<LibraryRoot[]>,
  listFolders: (rootId: number) => service("LibraryService").ListFolders(rootId) as Promise<FontFolder[]>,
  rescanRoot: (id: number) => service("LibraryService").RescanRoot(id) as Promise<ScanResult>,
  rescanAllRoots: () => service("LibraryService").RescanAllRoots() as Promise<number>,
  rescanFolder: (rootId: number, folderPath: string) => service("LibraryService").RescanFolder(rootId, folderPath) as Promise<ScanResult>,
  removeRoot: (id: number) => service("LibraryService").RemoveRoot(id),
  searchFonts: (query: FontQuery) => service("FontService").SearchFonts(query) as Promise<FontItem[]>,
  getFontDetail: (faceId: number) => service("FontService").GetFontDetail(faceId) as Promise<FontDetail>,
  setFavorite: (faceId: number, favorite: boolean) => service("FontService").SetFavorite(faceId, favorite),
  getPreview: (faceId: number, sampleText: string) => service("FontService").GetPreview(faceId, sampleText) as Promise<PreviewResponse>,
  revealInExplorer: (faceId: number) => service("FontService").RevealInExplorer(faceId),
  installFonts: (faceIds: number[], mode: "copy" | "link", scope = "user") =>
    service("InstallService").InstallFonts(faceIds, mode, scope) as Promise<OperationResult>,
  uninstallFonts: (faceIds: number[], deleteCopiedFiles = true) =>
    service("InstallService").UninstallFonts(faceIds, deleteCopiedFiles) as Promise<OperationResult>,
};
