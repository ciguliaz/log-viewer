export namespace main {
	
	export class Connection {
	    app: string;
	    hash: string;
	    dest: string;
	    packets: number;
	    route: string;
	    last_seen: string;
	
	    static createFrom(source: any = {}) {
	        return new Connection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.app = source["app"];
	        this.hash = source["hash"];
	        this.dest = source["dest"];
	        this.packets = source["packets"];
	        this.route = source["route"];
	        this.last_seen = source["last_seen"];
	    }
	}

}

