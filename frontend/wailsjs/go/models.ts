export namespace models {
	
	export class AppInfo {
	    name: string;
	    version: string;
	    dataDir: string;
	    cacheDir: string;
	    logDir: string;
	    databasePath: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.version = source["version"];
	        this.dataDir = source["dataDir"];
	        this.cacheDir = source["cacheDir"];
	        this.logDir = source["logDir"];
	        this.databasePath = source["databasePath"];
	    }
	}
	export class InstallRecord {
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
	
	    static createFrom(source: any = {}) {
	        return new InstallRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fileId = source["fileId"];
	        this.faceId = source["faceId"];
	        this.sourcePath = source["sourcePath"];
	        this.targetPath = source["targetPath"];
	        this.mode = source["mode"];
	        this.scope = source["scope"];
	        this.registryKey = source["registryKey"];
	        this.registryValueName = source["registryValueName"];
	        this.registryValueData = source["registryValueData"];
	        this.installedAt = source["installedAt"];
	        this.uninstalledAt = source["uninstalledAt"];
	        this.status = source["status"];
	        this.error = source["error"];
	    }
	}
	export class FontDetail {
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
	
	    static createFrom(source: any = {}) {
	        return new FontDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.faceId = source["faceId"];
	        this.fileId = source["fileId"];
	        this.rootId = source["rootId"];
	        this.rootPath = source["rootPath"];
	        this.path = source["path"];
	        this.fileName = source["fileName"];
	        this.format = source["format"];
	        this.family = source["family"];
	        this.style = source["style"];
	        this.fullName = source["fullName"];
	        this.postScriptName = source["postScriptName"];
	        this.weight = source["weight"];
	        this.italic = source["italic"];
	        this.isFavorite = source["isFavorite"];
	        this.isInstalled = source["isInstalled"];
	        this.previewSupported = source["previewSupported"];
	        this.status = source["status"];
	        this.error = source["error"];
	        this.updatedAt = source["updatedAt"];
	        this.size = source["size"];
	        this.modifiedAt = source["modifiedAt"];
	        this.hash = source["hash"];
	        this.sampleText = source["sampleText"];
	        this.manufacturer = source["manufacturer"];
	        this.designer = source["designer"];
	        this.license = source["license"];
	        this.version = source["version"];
	        this.glyphCount = source["glyphCount"];
	        this.installRecords = this.convertValues(source["installRecords"], InstallRecord);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class FontFolder {
	    rootId: number;
	    path: string;
	    name: string;
	    depth: number;
	    fontCount: number;
	
	    static createFrom(source: any = {}) {
	        return new FontFolder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootId = source["rootId"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.depth = source["depth"];
	        this.fontCount = source["fontCount"];
	    }
	}
	export class FontItem {
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
	
	    static createFrom(source: any = {}) {
	        return new FontItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.faceId = source["faceId"];
	        this.fileId = source["fileId"];
	        this.rootId = source["rootId"];
	        this.rootPath = source["rootPath"];
	        this.path = source["path"];
	        this.fileName = source["fileName"];
	        this.format = source["format"];
	        this.family = source["family"];
	        this.style = source["style"];
	        this.fullName = source["fullName"];
	        this.postScriptName = source["postScriptName"];
	        this.weight = source["weight"];
	        this.italic = source["italic"];
	        this.isFavorite = source["isFavorite"];
	        this.isInstalled = source["isInstalled"];
	        this.previewSupported = source["previewSupported"];
	        this.status = source["status"];
	        this.error = source["error"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class FontQuery {
	    query: string;
	    rootId: number;
	    folderPath: string;
	    folderRecursive: boolean;
	    favoritesOnly: boolean;
	    installedOnly: boolean;
	    limit: number;
	    offset: number;
	
	    static createFrom(source: any = {}) {
	        return new FontQuery(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.rootId = source["rootId"];
	        this.folderPath = source["folderPath"];
	        this.folderRecursive = source["folderRecursive"];
	        this.favoritesOnly = source["favoritesOnly"];
	        this.installedOnly = source["installedOnly"];
	        this.limit = source["limit"];
	        this.offset = source["offset"];
	    }
	}
	
	export class LibraryRoot {
	    id: number;
	    path: string;
	    name: string;
	    kind: string;
	    enabled: boolean;
	    createdAt: string;
	    updatedAt: string;
	    lastScanAt: string;
	    fontCount: number;
	    scanStatus: string;
	    scanTotal: number;
	    scanProcessed: number;
	
	    static createFrom(source: any = {}) {
	        return new LibraryRoot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.kind = source["kind"];
	        this.enabled = source["enabled"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	        this.lastScanAt = source["lastScanAt"];
	        this.fontCount = source["fontCount"];
	        this.scanStatus = source["scanStatus"];
	        this.scanTotal = source["scanTotal"];
	        this.scanProcessed = source["scanProcessed"];
	    }
	}
	export class OperationMessage {
	    faceId: number;
	    fileId: number;
	    level: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new OperationMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.faceId = source["faceId"];
	        this.fileId = source["fileId"];
	        this.level = source["level"];
	        this.message = source["message"];
	    }
	}
	export class OperationResult {
	    operation: string;
	    succeeded: number;
	    failed: number;
	    messages: OperationMessage[];
	
	    static createFrom(source: any = {}) {
	        return new OperationResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operation = source["operation"];
	        this.succeeded = source["succeeded"];
	        this.failed = source["failed"];
	        this.messages = this.convertValues(source["messages"], OperationMessage);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PreviewResponse {
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
	
	    static createFrom(source: any = {}) {
	        return new PreviewResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.faceId = source["faceId"];
	        this.fontFamily = source["fontFamily"];
	        this.css = source["css"];
	        this.fontUrl = source["fontUrl"];
	        this.sampleText = source["sampleText"];
	        this.previewSupported = source["previewSupported"];
	        this.message = source["message"];
	        this.cacheHit = source["cacheHit"];
	        this.byteSize = source["byteSize"];
	        this.glyphCount = source["glyphCount"];
	        this.missingRuneCount = source["missingRuneCount"];
	        this.fullBytes = source["fullBytes"];
	        this.subsetBytes = source["subsetBytes"];
	        this.fallback = source["fallback"];
	        this.fallbackReason = source["fallbackReason"];
	        this.reductionRatio = source["reductionRatio"];
	    }
	}
	export class ScanResult {
	    rootId: number;
	    total: number;
	    processed: number;
	    added: number;
	    updated: number;
	    failed: number;
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootId = source["rootId"];
	        this.total = source["total"];
	        this.processed = source["processed"];
	        this.added = source["added"];
	        this.updated = source["updated"];
	        this.failed = source["failed"];
	    }
	}
	export class ScanStatus {
	    rootId: number;
	    status: string;
	    total: number;
	    processed: number;
	    added: number;
	    updated: number;
	    failed: number;
	    message: string;
	    startedAt: string;
	    finishedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ScanStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rootId = source["rootId"];
	        this.status = source["status"];
	        this.total = source["total"];
	        this.processed = source["processed"];
	        this.added = source["added"];
	        this.updated = source["updated"];
	        this.failed = source["failed"];
	        this.message = source["message"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	    }
	}

}

