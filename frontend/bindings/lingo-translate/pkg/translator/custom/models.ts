// This file defines the TypeScript interface for custom.Definition


/**
 * Definition represents a declarative external translation provider configuration (JSON file)
 */
export class Definition {
    "name": string;
    "display_name"?: string;
    "description"?: string;
    "type"?: string;
    "base_url": string;
    "default_model": string;
    "available_models"?: string[];
    "api_key_env"?: string;
    "api_key"?: string;
    "headers"?: { [_: string]: string };
    "timeout_sec"?: number;

    /** Creates a new Definition instance. */
    constructor($$source: Partial<Definition> = {}) {
        if (!("name" in $$source)) {
            this["name"] = "";
        }
        if (!("base_url" in $$source)) {
            this["base_url"] = "";
        }
        if (!("default_model" in $$source)) {
            this["default_model"] = "";
        }

        Object.assign(this, $$source);
    }

    /**
     * Creates a new Definition instance from a string or object.
     */
    static createFrom($$source: any = {}): Definition {
        let $$parsedSource = typeof $$source === 'string' ? JSON.parse($$source) : $$source;
        return new Definition($$parsedSource as Partial<Definition>);
    }
}
