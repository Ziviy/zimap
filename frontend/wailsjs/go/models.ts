export namespace device {
	
	export class DeviceMap {
	    fileName: string;
	    filePath: string;
	    internalName: string;
	    sizeBytes: number;
	    sizeFormatted: string;
	    // Go type: time
	    modTime: any;
	    isCustom: boolean;
	    isPreinstalled: boolean;
	    signatureType?: string;
	    relativeDir: string;
	    objectId?: number;
	
	    static createFrom(source: any = {}) {
	        return new DeviceMap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fileName = source["fileName"];
	        this.filePath = source["filePath"];
	        this.internalName = source["internalName"];
	        this.sizeBytes = source["sizeBytes"];
	        this.sizeFormatted = source["sizeFormatted"];
	        this.modTime = this.convertValues(source["modTime"], null);
	        this.isCustom = source["isCustom"];
	        this.isPreinstalled = source["isPreinstalled"];
	        this.signatureType = source["signatureType"];
	        this.relativeDir = source["relativeDir"];
	        this.objectId = source["objectId"];
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
	export class MapFreshnessItem {
	    id: string;
	    mapName: string;
	    fileName: string;
	    status: string;
	    statusMessage: string;
	    localPath?: string;
	    localSize?: number;
	    localSizeStr?: string;
	    // Go type: time
	    localModTime?: any;
	    devicePath?: string;
	    deviceSize?: number;
	    deviceSizeStr?: string;
	    // Go type: time
	    deviceModTime?: any;
	    deviceDir?: string;
	    internalName?: string;
	    bytesToTransfer: number;
	    hasEnoughSpace: boolean;
	    daysOutdated?: number;
	    isPreinstalled: boolean;
	    signatureType?: string;
	
	    static createFrom(source: any = {}) {
	        return new MapFreshnessItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.mapName = source["mapName"];
	        this.fileName = source["fileName"];
	        this.status = source["status"];
	        this.statusMessage = source["statusMessage"];
	        this.localPath = source["localPath"];
	        this.localSize = source["localSize"];
	        this.localSizeStr = source["localSizeStr"];
	        this.localModTime = this.convertValues(source["localModTime"], null);
	        this.devicePath = source["devicePath"];
	        this.deviceSize = source["deviceSize"];
	        this.deviceSizeStr = source["deviceSizeStr"];
	        this.deviceModTime = this.convertValues(source["deviceModTime"], null);
	        this.deviceDir = source["deviceDir"];
	        this.internalName = source["internalName"];
	        this.bytesToTransfer = source["bytesToTransfer"];
	        this.hasEnoughSpace = source["hasEnoughSpace"];
	        this.daysOutdated = source["daysOutdated"];
	        this.isPreinstalled = source["isPreinstalled"];
	        this.signatureType = source["signatureType"];
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
	export class DeviceFreshnessReport {
	    mountPath: string;
	    deviceName: string;
	    // Go type: time
	    checkedAt: any;
	    items: MapFreshnessItem[];
	    preinstalledMaps?: DeviceMap[];
	    preinstalledCount: number;
	    upToDateCount: number;
	    updateAvailableCount: number;
	    notOnDeviceCount: number;
	    onlyOnDeviceCount: number;
	    totalSyncBytes: number;
	    totalSyncBytesStr: string;
	    deviceFreeBytes: number;
	    deviceFreeStr: string;
	
	    static createFrom(source: any = {}) {
	        return new DeviceFreshnessReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mountPath = source["mountPath"];
	        this.deviceName = source["deviceName"];
	        this.checkedAt = this.convertValues(source["checkedAt"], null);
	        this.items = this.convertValues(source["items"], MapFreshnessItem);
	        this.preinstalledMaps = this.convertValues(source["preinstalledMaps"], DeviceMap);
	        this.preinstalledCount = source["preinstalledCount"];
	        this.upToDateCount = source["upToDateCount"];
	        this.updateAvailableCount = source["updateAvailableCount"];
	        this.notOnDeviceCount = source["notOnDeviceCount"];
	        this.onlyOnDeviceCount = source["onlyOnDeviceCount"];
	        this.totalSyncBytes = source["totalSyncBytes"];
	        this.totalSyncBytesStr = source["totalSyncBytesStr"];
	        this.deviceFreeBytes = source["deviceFreeBytes"];
	        this.deviceFreeStr = source["deviceFreeStr"];
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
	
	export class StorageInfo {
	    totalBytes: number;
	    freeBytes: number;
	    usedBytes: number;
	    usedPercentage: number;
	    formattedTotal: string;
	    formattedFree: string;
	    formattedUsed: string;
	
	    static createFrom(source: any = {}) {
	        return new StorageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalBytes = source["totalBytes"];
	        this.freeBytes = source["freeBytes"];
	        this.usedBytes = source["usedBytes"];
	        this.usedPercentage = source["usedPercentage"];
	        this.formattedTotal = source["formattedTotal"];
	        this.formattedFree = source["formattedFree"];
	        this.formattedUsed = source["formattedUsed"];
	    }
	}
	export class GarminDevice {
	    mountPath: string;
	    volumeName: string;
	    modelName: string;
	    displayName: string;
	    unitId: string;
	    softwareVersion: string;
	    storage: StorageInfo;
	    customMaps: DeviceMap[];
	    preinstalledMaps?: DeviceMap[];
	    totalMapsCount: number;
	    // Go type: time
	    connectedAt: any;
	    isMtp: boolean;
	    storageId?: number;
	    mtpConflict?: string;
	
	    static createFrom(source: any = {}) {
	        return new GarminDevice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mountPath = source["mountPath"];
	        this.volumeName = source["volumeName"];
	        this.modelName = source["modelName"];
	        this.displayName = source["displayName"];
	        this.unitId = source["unitId"];
	        this.softwareVersion = source["softwareVersion"];
	        this.storage = this.convertValues(source["storage"], StorageInfo);
	        this.customMaps = this.convertValues(source["customMaps"], DeviceMap);
	        this.preinstalledMaps = this.convertValues(source["preinstalledMaps"], DeviceMap);
	        this.totalMapsCount = source["totalMapsCount"];
	        this.connectedAt = this.convertValues(source["connectedAt"], null);
	        this.isMtp = source["isMtp"];
	        this.storageId = source["storageId"];
	        this.mtpConflict = source["mtpConflict"];
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
	
	
	export class USBStatus {
	    garminUsbAttached: boolean;
	    deviceDescription: string;
	    needsMtpNotice: boolean;
	    isGarminMode: boolean;
	    mtpConflict?: string;
	
	    static createFrom(source: any = {}) {
	        return new USBStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.garminUsbAttached = source["garminUsbAttached"];
	        this.deviceDescription = source["deviceDescription"];
	        this.needsMtpNotice = source["needsMtpNotice"];
	        this.isGarminMode = source["isGarminMode"];
	        this.mtpConflict = source["mtpConflict"];
	    }
	}

}

export namespace download {
	
	export class ProxyConfig {
	    useSystemProxy: boolean;
	    customProxy?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxyConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.useSystemProxy = source["useSystemProxy"];
	        this.customProxy = source["customProxy"];
	    }
	}

}

export namespace main {
	
	export class RegionData {
	    id: string;
	    parentId: string;
	    name: string;
	    pbfUrl: string;
	    children: RegionData[];
	
	    static createFrom(source: any = {}) {
	        return new RegionData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.parentId = source["parentId"];
	        this.name = source["name"];
	        this.pbfUrl = source["pbfUrl"];
	        this.children = this.convertValues(source["children"], RegionData);
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
	export class CatalogData {
	    roots: RegionData[];
	
	    static createFrom(source: any = {}) {
	        return new CatalogData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.roots = this.convertValues(source["roots"], RegionData);
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
	export class MapMetadata {
	    name: string;
	    sourceType: string;
	    sourceURL: string;
	    regionID?: string;
	    heapSize?: string;
	    detailLevel?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	
	    static createFrom(source: any = {}) {
	        return new MapMetadata(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.sourceType = source["sourceType"];
	        this.sourceURL = source["sourceURL"];
	        this.regionID = source["regionID"];
	        this.heapSize = source["heapSize"];
	        this.detailLevel = source["detailLevel"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
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
	export class OutputItem {
	    name: string;
	    path: string;
	    size: number;
	    // Go type: time
	    modTime: any;
	    sourceUrl?: string;
	    meta?: MapMetadata;
	
	    static createFrom(source: any = {}) {
	        return new OutputItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.size = source["size"];
	        this.modTime = this.convertValues(source["modTime"], null);
	        this.sourceUrl = source["sourceUrl"];
	        this.meta = this.convertValues(source["meta"], MapMetadata);
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
	
	export class StorageMetrics {
	    cacheSize: number;
	    tempSize: number;
	    mapsSize: number;
	    totalSize: number;
	
	    static createFrom(source: any = {}) {
	        return new StorageMetrics(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.cacheSize = source["cacheSize"];
	        this.tempSize = source["tempSize"];
	        this.mapsSize = source["mapsSize"];
	        this.totalSize = source["totalSize"];
	    }
	}
	export class ToolDetail {
	    installed: boolean;
	    embedded: boolean;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.embedded = source["embedded"];
	        this.path = source["path"];
	    }
	}
	export class ToolsStatus {
	    installed: boolean;
	    java: ToolDetail;
	    mkgmap: ToolDetail;
	    splitter: ToolDetail;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ToolsStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.installed = source["installed"];
	        this.java = this.convertValues(source["java"], ToolDetail);
	        this.mkgmap = this.convertValues(source["mkgmap"], ToolDetail);
	        this.splitter = this.convertValues(source["splitter"], ToolDetail);
	        this.error = source["error"];
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
	export class UpdateInfo {
	    currentVersion: string;
	    version: string;
	    releaseNotes: string;
	    downloadURL: string;
	    assetURL: string;
	    assetName: string;
	    assetSize: number;
	    isNewer: boolean;
	    canAutoUpdate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currentVersion = source["currentVersion"];
	        this.version = source["version"];
	        this.releaseNotes = source["releaseNotes"];
	        this.downloadURL = source["downloadURL"];
	        this.assetURL = source["assetURL"];
	        this.assetName = source["assetName"];
	        this.assetSize = source["assetSize"];
	        this.isNewer = source["isNewer"];
	        this.canAutoUpdate = source["canAutoUpdate"];
	    }
	}

}

export namespace provider {
	
	export class Info {
	    id: string;
	    name: string;
	    description: string;
	    homepage: string;
	
	    static createFrom(source: any = {}) {
	        return new Info(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.homepage = source["homepage"];
	    }
	}

}

