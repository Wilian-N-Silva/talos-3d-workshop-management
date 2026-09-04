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
}

interface NativeApp {
    GetServerConnection(): Promise<ServerConnection>;
    SaveServerConnection(baseURL: string): Promise<ServerConnection>;
    TestServerConnection(baseURL: string): Promise<ConnectionTestResult>;
    Login(emailOrUsername: string, password: string): Promise<AuthenticationState>;
    GetAuthenticationState(): Promise<AuthenticationState>;
    Logout(): Promise<void>;
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
