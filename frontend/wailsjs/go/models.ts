export namespace backend {
	
	export class bencodeInfo {
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new bencodeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	    }
	}
	export class BencodeTorrent {
	    announce: string;
	    info: bencodeInfo;
	
	    static createFrom(source: any = {}) {
	        return new BencodeTorrent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.announce = source["announce"];
	        this.info = this.convertValues(source["info"], bencodeInfo);
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

