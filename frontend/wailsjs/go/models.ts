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
	export class DropResult {
	    path: string;
	    name: string;
	    files: FileInfo[];
	
	    static createFrom(source: any = {}) {
	        return new DropResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.files = this.convertValues(source["files"], FileInfo);
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
	
	export class LogEntry {
	    id: string;
	    date: string;
	    time: string;
	    ms: string;
	    tz: string;
	    level: string;
	    tag: string;
	    message: string;
	    raw: string;
	    endDate: string;
	    endTime: string;
	    endMs: string;
	    endTz: string;
	    count: number;
	    lineNum: number;
	
	    static createFrom(source: any = {}) {
	        return new LogEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.date = source["date"];
	        this.time = source["time"];
	        this.ms = source["ms"];
	        this.tz = source["tz"];
	        this.level = source["level"];
	        this.tag = source["tag"];
	        this.message = source["message"];
	        this.raw = source["raw"];
	        this.endDate = source["endDate"];
	        this.endTime = source["endTime"];
	        this.endMs = source["endMs"];
	        this.endTz = source["endTz"];
	        this.count = source["count"];
	        this.lineNum = source["lineNum"];
	    }
	}

}

