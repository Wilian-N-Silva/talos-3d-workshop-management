export interface ServerConnection {
    server_base_url: string;
}

export interface ConnectionTestResult {
    server_base_url: string;
    desktop_version: string;
    api_version: string;
    server_version: string;
    workshop_name: string;
    minimum_desktop_version: string;
    compatible: boolean;
    compatibility_issue: '' | 'api_version_mismatch' | 'desktop_update_required';
}

export interface AuthenticationState {
    authenticated: boolean;
    user_id?: string;
    user_name?: string;
    email_or_username?: string;
    expires_at?: string;
    role?: string;
    permissions?: string[];
}

export interface WorkshopBranding {
    workshop_name: string;
    logo_data_url?: string;
}

export interface ShellContext {
    authentication: AuthenticationState;
    workshop_name?: string;
    logo_data_url?: string;
    default_theme?: ThemeMode;
}

export type ThemeMode = 'light' | 'dark' | 'system';
export type CatalogPurpose = 'product' | 'prototype' | 'tooling' | 'test' | 'internal' | 'personal';
export type CatalogStatus = 'active' | 'archived';

export interface CatalogItem {
    id: string;
    name: string;
    sku?: string | null;
    description: string;
    purpose: CatalogPurpose;
    sellable: boolean;
    tags: string[];
    status: CatalogStatus;
    created_at: string;
    updated_at: string;
}

export interface CatalogItemInput {
    name: string;
    sku?: string | null;
    description: string;
    purpose: CatalogPurpose;
    sellable: boolean;
    tags: string[];
    status: CatalogStatus;
}

export interface CatalogPage {
    items: CatalogItem[];
    pagination: {limit: number; offset: number; total: number};
}

export interface CatalogPart {
    id: string;
    catalog_item_id: string;
    name: string;
    quantity: number;
    notes: string;
    created_at: string;
    updated_at: string;
}

export interface CatalogPartInput {
    name: string;
    quantity: number;
    notes: string;
}

export type DesignOrigin = 'original' | 'customer' | 'remix' | 'third_party' | 'unknown';
export type DesignFileRole = 'source' | 'mesh' | 'print' | 'preview' | 'documentation' | 'other';

export interface DesignFile {
    file_id: string;
    role: DesignFileRole;
    original_name: string;
    content_type: string;
    size_bytes: number;
    sha256: string;
    created_at: string;
}

export interface DesignVersion {
    id: string;
    catalog_part_id: string;
    version: string;
    notes: string;
    origin: DesignOrigin;
    source_url?: string | null;
    original_author: string;
    license_name: string;
    commercial_use_allowed?: boolean | null;
    attribution_required: boolean;
    attribution_text: string;
    created_by: string;
    created_at: string;
    files: DesignFile[];
}

export interface DesignVersionInput {
    version: string;
    notes: string;
    origin: DesignOrigin;
    source_url?: string | null;
    original_author: string;
    license_name: string;
    commercial_use_allowed?: boolean | null;
    attribution_required: boolean;
    attribution_text: string;
}

interface NativeApp {
    GetServerConnection(): Promise<ServerConnection>;
    SaveServerConnection(baseURL: string): Promise<ServerConnection>;
    TestServerConnection(baseURL: string): Promise<ConnectionTestResult>;
    Login(emailOrUsername: string, password: string): Promise<AuthenticationState>;
    GetAuthenticationState(): Promise<AuthenticationState>;
    Logout(): Promise<void>;
    GetWorkshopBranding(): Promise<WorkshopBranding>;
    LoadShell(): Promise<ShellContext>;
    ListCatalogItems(): Promise<CatalogPage>;
    CreateCatalogItem(input: CatalogItemInput): Promise<CatalogItem>;
    UpdateCatalogItem(id: string, input: CatalogItemInput): Promise<CatalogItem>;
    ListCatalogParts(itemID: string): Promise<CatalogPart[]>;
    CreateCatalogPart(itemID: string, input: CatalogPartInput): Promise<CatalogPart>;
    ListDesignVersions(partID: string): Promise<DesignVersion[]>;
    CreateDesignVersion(partID: string, input: DesignVersionInput): Promise<DesignVersion>;
    AttachDesignFile(versionID: string, fileID: string, role: DesignFileRole): Promise<DesignFile>;
}

declare global {
    interface Window {
        go?: {
            desktopapp?: {
                App?: NativeApp;
            };
        };
    }
}

function app(): NativeApp {
    const nativeApp = window.go?.desktopapp?.App;
    if (!nativeApp) {
        throw new Error('Native desktop bridge is unavailable');
    }
    return nativeApp;
}

export async function getServerConnection(): Promise<ServerConnection> {
    return app().GetServerConnection();
}

export async function saveServerConnection(baseURL: string): Promise<ServerConnection> {
    return app().SaveServerConnection(baseURL);
}

export async function testServerConnection(baseURL: string): Promise<ConnectionTestResult> {
    return app().TestServerConnection(baseURL);
}

export async function login(emailOrUsername: string, password: string): Promise<AuthenticationState> {
    return app().Login(emailOrUsername, password);
}

export async function getAuthenticationState(): Promise<AuthenticationState> {
    return app().GetAuthenticationState();
}

export async function logout(): Promise<void> {
    return app().Logout();
}

export async function getWorkshopBranding(): Promise<WorkshopBranding> {
    return app().GetWorkshopBranding();
}

export async function loadShell(): Promise<ShellContext> {
    return app().LoadShell();
}

export async function listCatalogItems(): Promise<CatalogPage> {
    return app().ListCatalogItems();
}

export async function createCatalogItem(input: CatalogItemInput): Promise<CatalogItem> {
    return app().CreateCatalogItem(input);
}

export async function updateCatalogItem(id: string, input: CatalogItemInput): Promise<CatalogItem> {
    return app().UpdateCatalogItem(id, input);
}

export async function listCatalogParts(itemID: string): Promise<CatalogPart[]> {
    return app().ListCatalogParts(itemID);
}

export async function createCatalogPart(itemID: string, input: CatalogPartInput): Promise<CatalogPart> {
    return app().CreateCatalogPart(itemID, input);
}

export async function listDesignVersions(partID: string): Promise<DesignVersion[]> {
    return app().ListDesignVersions(partID);
}

export async function createDesignVersion(partID: string, input: DesignVersionInput): Promise<DesignVersion> {
    return app().CreateDesignVersion(partID, input);
}

export async function attachDesignFile(versionID: string, fileID: string, role: DesignFileRole): Promise<DesignFile> {
    return app().AttachDesignFile(versionID, fileID, role);
}
