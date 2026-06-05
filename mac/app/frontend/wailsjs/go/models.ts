export namespace config {
	
	export class BlockedEntry {
	    domain: string;
	    count: number;
	    // Go type: time
	    lastSeen: any;
	
	    static createFrom(source: any = {}) {
	        return new BlockedEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.domain = source["domain"];
	        this.count = source["count"];
	        this.lastSeen = this.convertValues(source["lastSeen"], null);
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

}

export namespace main {
	
	export class AdvancedSettings {
	    disableDelayHours: number;
	    blockedMessage: string;
	
	    static createFrom(source: any = {}) {
	        return new AdvancedSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.disableDelayHours = source["disableDelayHours"];
	        this.blockedMessage = source["blockedMessage"];
	    }
	}
	export class BlocklistData {
	    userAdded: string[];
	    builtInDomains: number;
	    builtInUrls: number;
	
	    static createFrom(source: any = {}) {
	        return new BlocklistData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userAdded = source["userAdded"];
	        this.builtInDomains = source["builtInDomains"];
	        this.builtInUrls = source["builtInUrls"];
	    }
	}
	export class ContentSettings {
	    blockAdultContent: boolean;
	    blockImageSearch: boolean;
	    blockYouTube: boolean;
	    safeSearch: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ContentSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.blockAdultContent = source["blockAdultContent"];
	        this.blockImageSearch = source["blockImageSearch"];
	        this.blockYouTube = source["blockYouTube"];
	        this.safeSearch = source["safeSearch"];
	    }
	}
	export class DisableDelayStatus {
	    delayHours: number;
	    requestPending: boolean;
	    readyToDisable: boolean;
	    remainingSeconds: number;
	
	    static createFrom(source: any = {}) {
	        return new DisableDelayStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.delayHours = source["delayHours"];
	        this.requestPending = source["requestPending"];
	        this.readyToDisable = source["readyToDisable"];
	        this.remainingSeconds = source["remainingSeconds"];
	    }
	}
	export class FocusModeStatus {
	    active: boolean;
	    remaining: number;
	
	    static createFrom(source: any = {}) {
	        return new FocusModeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.active = source["active"];
	        this.remaining = source["remaining"];
	    }
	}
	export class KeywordsData {
	    userAdded: string[];
	    builtInCount: number;
	
	    static createFrom(source: any = {}) {
	        return new KeywordsData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.userAdded = source["userAdded"];
	        this.builtInCount = source["builtInCount"];
	    }
	}
	export class ProxySettings {
	    proxyPort: number;
	    autoStart: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ProxySettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxyPort = source["proxyPort"];
	        this.autoStart = source["autoStart"];
	    }
	}
	export class Status {
	    proxyRunning: boolean;
	    layer1Active: boolean;
	    blockedToday: number;
	    totalBlocked: number;
	    proxyPort: number;
	    topBlocked: config.BlockedEntry[];
	    dbDomains: number;
	    dbUrls: number;
	    dbKeywords: number;
	    inFocusMode: boolean;
	    focusRemaining: number;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proxyRunning = source["proxyRunning"];
	        this.layer1Active = source["layer1Active"];
	        this.blockedToday = source["blockedToday"];
	        this.totalBlocked = source["totalBlocked"];
	        this.proxyPort = source["proxyPort"];
	        this.topBlocked = this.convertValues(source["topBlocked"], config.BlockedEntry);
	        this.dbDomains = source["dbDomains"];
	        this.dbUrls = source["dbUrls"];
	        this.dbKeywords = source["dbKeywords"];
	        this.inFocusMode = source["inFocusMode"];
	        this.focusRemaining = source["focusRemaining"];
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

}

