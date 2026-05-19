export namespace config {
	
	export class EnvSelectOption {
	    value: string;
	    label: string;
	
	    static createFrom(source: any = {}) {
	        return new EnvSelectOption(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.label = source["label"];
	    }
	}
	export class EnvItem {
	    key: string;
	    label: string;
	    value: string;
	    default_value: string;
	    type: string;
	    group: string;
	    description: string;
	    options?: EnvSelectOption[];
	    depends_on?: string;
	    depends_value?: string;
	    restart_required: boolean;
	
	    static createFrom(source: any = {}) {
	        return new EnvItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.label = source["label"];
	        this.value = source["value"];
	        this.default_value = source["default_value"];
	        this.type = source["type"];
	        this.group = source["group"];
	        this.description = source["description"];
	        this.options = this.convertValues(source["options"], EnvSelectOption);
	        this.depends_on = source["depends_on"];
	        this.depends_value = source["depends_value"];
	        this.restart_required = source["restart_required"];
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

export namespace handler {
	
	export class ActiveToolCall {
	    id: string;
	    name: string;
	    arguments: string;
	
	    static createFrom(source: any = {}) {
	        return new ActiveToolCall(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.arguments = source["arguments"];
	    }
	}
	export class ActiveRequest {
	    request_id: string;
	    provider_id: number;
	    provider: string;
	    model: string;
	    request_body: string;
	    response_content: string;
	    tool_calls: ActiveToolCall[];
	    status: string;
	    // Go type: time
	    start_time: any;
	    protocol: string;
	    client_ip: string;
	
	    static createFrom(source: any = {}) {
	        return new ActiveRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.request_id = source["request_id"];
	        this.provider_id = source["provider_id"];
	        this.provider = source["provider"];
	        this.model = source["model"];
	        this.request_body = source["request_body"];
	        this.response_content = source["response_content"];
	        this.tool_calls = this.convertValues(source["tool_calls"], ActiveToolCall);
	        this.status = source["status"];
	        this.start_time = this.convertValues(source["start_time"], null);
	        this.protocol = source["protocol"];
	        this.client_ip = source["client_ip"];
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
	
	export class CodeBuddyResultVO {
	    message: string;
	    path: string;
	    exists: boolean;
	    added: boolean;
	    models: number;
	
	    static createFrom(source: any = {}) {
	        return new CodeBuddyResultVO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.path = source["path"];
	        this.exists = source["exists"];
	        this.added = source["added"];
	        this.models = source["models"];
	    }
	}
	export class HourlyStatBreakdownVO {
	    hour: number;
	    provider_id: number;
	    provider_name: string;
	    input_tokens: number;
	    output_tokens: number;
	    total_tokens: number;
	
	    static createFrom(source: any = {}) {
	        return new HourlyStatBreakdownVO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hour = source["hour"];
	        this.provider_id = source["provider_id"];
	        this.provider_name = source["provider_name"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.total_tokens = source["total_tokens"];
	    }
	}
	export class LogEntryVO {
	    time: string;
	    level: string;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new LogEntryVO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.time = source["time"];
	        this.level = source["level"];
	        this.message = source["message"];
	    }
	}
	export class ProviderCreateVO {
	    name: string;
	    auto_suffix: boolean;
	    url_suffix: string;
	    base_url: string;
	    api_key: string;
	    model: string;
	    alias: string;
	    extra_params: string;
	
	    static createFrom(source: any = {}) {
	        return new ProviderCreateVO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.auto_suffix = source["auto_suffix"];
	        this.url_suffix = source["url_suffix"];
	        this.base_url = source["base_url"];
	        this.api_key = source["api_key"];
	        this.model = source["model"];
	        this.alias = source["alias"];
	        this.extra_params = source["extra_params"];
	    }
	}
	export class ProviderUpdateVO {
	    name: string;
	    auto_suffix: boolean;
	    url_suffix: string;
	    base_url: string;
	    api_key: string;
	    model: string;
	    alias: string;
	    extra_params: string;
	
	    static createFrom(source: any = {}) {
	        return new ProviderUpdateVO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.auto_suffix = source["auto_suffix"];
	        this.url_suffix = source["url_suffix"];
	        this.base_url = source["base_url"];
	        this.api_key = source["api_key"];
	        this.model = source["model"];
	        this.alias = source["alias"];
	        this.extra_params = source["extra_params"];
	    }
	}
	export class ProviderVO {
	    id: number;
	    name: string;
	    auto_suffix: boolean;
	    url_suffix: string;
	    base_url: string;
	    api_key: string;
	    model: string;
	    alias: string;
	    extra_params: string;
	    created_at: string;
	    updated_at: string;
	
	    static createFrom(source: any = {}) {
	        return new ProviderVO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.auto_suffix = source["auto_suffix"];
	        this.url_suffix = source["url_suffix"];
	        this.base_url = source["base_url"];
	        this.api_key = source["api_key"];
	        this.model = source["model"];
	        this.alias = source["alias"];
	        this.extra_params = source["extra_params"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	    }
	}
	export class ProxyStatusVO {
	    status: string;
	    port: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ProxyStatusVO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.port = source["port"];
	        this.error = source["error"];
	    }
	}
	export class RequestLogDetailVO {
	    id: number;
	    provider_id: number;
	    provider_name: string;
	    model: string;
	    input_tokens: number;
	    output_tokens: number;
	    total_tokens: number;
	    cached_tokens: number;
	    status: string;
	    error_message: string;
	    duration: number;
	    created_at: string;
	    response_content: string;
	    thinking_content: string;
	    request_body: string;
	    response_body: string;
	
	    static createFrom(source: any = {}) {
	        return new RequestLogDetailVO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.provider_id = source["provider_id"];
	        this.provider_name = source["provider_name"];
	        this.model = source["model"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.total_tokens = source["total_tokens"];
	        this.cached_tokens = source["cached_tokens"];
	        this.status = source["status"];
	        this.error_message = source["error_message"];
	        this.duration = source["duration"];
	        this.created_at = source["created_at"];
	        this.response_content = source["response_content"];
	        this.thinking_content = source["thinking_content"];
	        this.request_body = source["request_body"];
	        this.response_body = source["response_body"];
	    }
	}
	export class RequestLogVO {
	    id: number;
	    provider_id: number;
	    provider_name: string;
	    model: string;
	    input_tokens: number;
	    output_tokens: number;
	    total_tokens: number;
	    cached_tokens: number;
	    status: string;
	    error_message: string;
	    duration: number;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new RequestLogVO(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.provider_id = source["provider_id"];
	        this.provider_name = source["provider_name"];
	        this.model = source["model"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.total_tokens = source["total_tokens"];
	        this.cached_tokens = source["cached_tokens"];
	        this.status = source["status"];
	        this.error_message = source["error_message"];
	        this.duration = source["duration"];
	        this.created_at = source["created_at"];
	    }
	}

}

export namespace model {
	
	export class HourlyStatsResult {
	    hour: number;
	    request_count: number;
	    total_tokens: number;
	    input_tokens: number;
	    output_tokens: number;
	    cached_tokens: number;
	
	    static createFrom(source: any = {}) {
	        return new HourlyStatsResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.hour = source["hour"];
	        this.request_count = source["request_count"];
	        this.total_tokens = source["total_tokens"];
	        this.input_tokens = source["input_tokens"];
	        this.output_tokens = source["output_tokens"];
	        this.cached_tokens = source["cached_tokens"];
	    }
	}
	export class TokenStats {
	    date: string;
	    total_input_tokens: number;
	    total_output_tokens: number;
	    total_tokens: number;
	    total_cached_tokens: number;
	    request_count: number;
	
	    static createFrom(source: any = {}) {
	        return new TokenStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.total_input_tokens = source["total_input_tokens"];
	        this.total_output_tokens = source["total_output_tokens"];
	        this.total_tokens = source["total_tokens"];
	        this.total_cached_tokens = source["total_cached_tokens"];
	        this.request_count = source["request_count"];
	    }
	}

}

