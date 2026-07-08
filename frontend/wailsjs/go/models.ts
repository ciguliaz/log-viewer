export namespace main {
	
	export class FileInfo {
	    name: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	    }
	}
	export class LogEntry {
	    id: string;
	    time: string;
	    endTime: string;
	    count: number;
	    level: string;
	    tag: string;
	    message: string;
	    raw: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.time = source["time"];
	        this.endTime = source["endTime"];
	        this.count = source["count"];
	        this.level = source["level"];
	        this.tag = source["tag"];
	        this.message = source["message"];
	        this.raw = source["raw"];
	    }
	}

}

