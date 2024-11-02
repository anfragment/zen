export namespace cfg {
	
	export class Config {
	    // Go type: struct { FilterLists []cfg
	    filter: any;
	    // Go type: struct { CAInstalled bool "json:\"caInstalled\"" }
	    certmanager: any;
	    // Go type: struct { Port int "json:\"port\""; IgnoredHosts []string "json:\"ignoredHosts\"" }
	    proxy: any;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.filter = this.convertValues(source["filter"], Object);
	        this.certmanager = this.convertValues(source["certmanager"], Object);
	        this.proxy = this.convertValues(source["proxy"], Object);
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
	export class FilterList {
	    name: string;
	    type: string;
	    url: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FilterList(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.type = source["type"];
	        this.url = source["url"];
	        this.enabled = source["enabled"];
	    }
	}

}

