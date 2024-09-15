export namespace backend {
	
	export class Torrent {
	    id: number;
	    torrentName: string;
	    fileNames: string[];
	    progress: number;
	    isMultiFile: boolean;
	    totalLength: number;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new Torrent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.torrentName = source["torrentName"];
	        this.fileNames = source["fileNames"];
	        this.progress = source["progress"];
	        this.isMultiFile = source["isMultiFile"];
	        this.totalLength = source["totalLength"];
	        this.status = source["status"];
	    }
	}

}

