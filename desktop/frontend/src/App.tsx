import {FormEvent, useCallback, useEffect, useRef, useState} from 'react';
import {WindowSetTitle} from '../wailsjs/runtime/runtime';
import './App.css';
import {CatalogWorkspace} from './CatalogWorkspace';
import {InventoryWorkspace} from './InventoryWorkspace';
import {JobWorkspace} from './JobWorkspace';
import {LaborWorkspace} from './LaborWorkspace';
import {
    AuthenticationState,
    ConnectionTestResult,
    ShellContext,
    ThemeMode,
    WorkshopBranding,
    getAuthenticationState,
    getServerConnection,
    getWorkshopBranding,
    loadShell,
    login,
    logout,
    saveServerConnection,
    testServerConnection,
} from './native';
import {PermissionGate, PermissionProvider} from './permissions';

type Feedback = {tone: 'success' | 'warning' | 'error'; message: string};
type Screen = 'connection' | 'login' | 'shell';
const themeStorageKey = 'talos.desktop.theme';

function storedTheme(): ThemeMode | null {
    const value = localStorage.getItem(themeStorageKey);
    return value === 'light' || value === 'dark' || value === 'system' ? value : null;
}

function errorMessage(error: unknown): string {
    if (error instanceof Error) return error.message;
    return typeof error === 'string' ? error : 'Não foi possível concluir a operação.';
}

function compatibilityMessage(result: ConnectionTestResult): Feedback {
    if (result.compatible) return {tone: 'success', message: `Conectado a ${result.workshop_name}.`};
    if (result.compatibility_issue === 'desktop_update_required') {
        return {tone: 'warning', message: `Este servidor requer o desktop ${result.minimum_desktop_version} ou mais recente.`};
    }
    return {tone: 'warning', message: `A API ${result.api_version} não é compatível com este aplicativo.`};
}

function Brand({branding}: {branding: WorkshopBranding}) {
    return (
        <div className="brand">
            {branding.logo_data_url
                ? <img className="brand__logo" src={branding.logo_data_url} alt="" />
                : <span className="brand__fallback" aria-hidden="true">T</span>}
            <div>
                <p className="connection-card__eyebrow">Talos · Oficina 3D</p>
                <strong>{branding.workshop_name || 'Oficina 3D'}</strong>
            </div>
        </div>
    );
}

function App() {
    const initialTheme = storedTheme();
    const themeOverridden = useRef(initialTheme !== null);
    const [theme, setTheme] = useState<ThemeMode>(initialTheme ?? 'system');
    const [screen, setScreen] = useState<Screen>('connection');
    const [baseURL, setBaseURL] = useState('');
    const [loading, setLoading] = useState(true);
    const [busyAction, setBusyAction] = useState<'save' | 'test' | 'login' | 'logout' | null>(null);
    const [feedback, setFeedback] = useState<Feedback | null>(null);
    const [connection, setConnection] = useState<ConnectionTestResult | null>(null);
    const [emailOrUsername, setEmailOrUsername] = useState('');
    const [password, setPassword] = useState('');
    const [authentication, setAuthentication] = useState<AuthenticationState | null>(null);
    const [branding, setBranding] = useState<WorkshopBranding>({workshop_name: 'Oficina 3D'});

    useEffect(() => {
        document.documentElement.dataset.theme = theme;
        if (themeOverridden.current) localStorage.setItem(themeStorageKey, theme);
    }, [theme]);

    const applyShell = useCallback((context: ShellContext): boolean => {
        if (!context.authentication.authenticated) return false;
        setAuthentication(context.authentication);
        setBranding({workshop_name: context.workshop_name || 'Oficina 3D', logo_data_url: context.logo_data_url});
        if (!themeOverridden.current && context.default_theme) setTheme(context.default_theme);
        WindowSetTitle(`${context.workshop_name || 'Oficina 3D'} — Talos`);
        setScreen('shell');
        return true;
    }, []);

    useEffect(() => {
        void Promise.all([getServerConnection(), getAuthenticationState()])
            .then(async ([configuration, restored]) => {
                setBaseURL(configuration.server_base_url);
                if (!configuration.server_base_url) return;
                try {
                    const workshopBranding = await getWorkshopBranding();
                    setBranding(workshopBranding);
                    WindowSetTitle(`${workshopBranding.workshop_name} — Talos`);
                } catch {
                    setBranding({workshop_name: 'Oficina 3D'});
                }
                if (restored.authenticated) {
                    const context = await loadShell();
                    if (applyShell(context)) return;
                }
                setScreen('login');
            })
            .catch((error: unknown) => setFeedback({tone: 'error', message: errorMessage(error)}))
            .finally(() => setLoading(false));
    }, [applyShell]);

    async function testConnection() {
        setBusyAction('test');
        setFeedback(null);
        setConnection(null);
        try {
            const result = await testServerConnection(baseURL);
            setBaseURL(result.server_base_url);
            setConnection(result);
            setFeedback(compatibilityMessage(result));
        } catch (error: unknown) {
            setFeedback({tone: 'error', message: errorMessage(error)});
        } finally {
            setBusyAction(null);
        }
    }

    async function saveConnection(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setBusyAction('save');
        setFeedback(null);
        try {
            const configuration = await saveServerConnection(baseURL);
            setBaseURL(configuration.server_base_url);
            setConnection(null);
            try {
                const workshopBranding = await getWorkshopBranding();
                setBranding(workshopBranding);
                WindowSetTitle(`${workshopBranding.workshop_name} — Talos`);
            } catch {
                setBranding({workshop_name: 'Oficina 3D'});
            }
            setScreen('login');
        } catch (error: unknown) {
            setFeedback({tone: 'error', message: errorMessage(error)});
        } finally {
            setBusyAction(null);
        }
    }

    async function submitLogin(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setBusyAction('login');
        setFeedback(null);
        try {
            await login(emailOrUsername, password);
            setPassword('');
            const context = await loadShell();
            if (!applyShell(context)) {
                setFeedback({tone: 'error', message: 'A sessão não pôde ser validada.'});
            }
        } catch (error: unknown) {
            setPassword('');
            setFeedback({tone: 'error', message: errorMessage(error)});
        } finally {
            setBusyAction(null);
        }
    }

    async function submitLogout() {
        setBusyAction('logout');
        setFeedback(null);
        try {
            await logout();
            setAuthentication(null);
            WindowSetTitle(`${branding.workshop_name || 'Oficina 3D'} — Talos`);
            setScreen('login');
        } catch (error: unknown) {
            setFeedback({tone: 'error', message: errorMessage(error)});
        } finally {
            setBusyAction(null);
        }
    }

    function changeTheme(value: ThemeMode) {
        themeOverridden.current = true;
        setTheme(value);
    }

    if (loading) return <main className="connection-page"><p>Carregando…</p></main>;

    if (screen === 'shell' && authentication) {
        return (
            <PermissionProvider permissions={authentication.permissions ?? []}>
                <main className="app-shell">
                    <header className="app-shell__header">
                        <Brand branding={branding} />
                        <div className="app-shell__account">
                            <PermissionGate permission="settings.manage">
                                <span className="permission-badge">Configurações permitidas</span>
                            </PermissionGate>
                            <label className="theme-picker">
                                <span>Tema</span>
                                <select value={theme} onChange={(event) => changeTheme(event.target.value as ThemeMode)}>
                                    <option value="light">Claro</option>
                                    <option value="dark">Escuro</option>
                                    <option value="system">Sistema</option>
                                </select>
                            </label>
                            <span>{authentication.user_name}</span>
                            <button className="button button--secondary" disabled={busyAction === 'logout'} onClick={() => void submitLogout()}>
                                {busyAction === 'logout' ? 'Saindo…' : 'Sair'}
                            </button>
                        </div>
                    </header>
                    <PermissionGate permission="catalog.read">
                        <CatalogWorkspace canWrite={(authentication.permissions ?? []).includes('catalog.write')} />
                    </PermissionGate>
                    <PermissionGate permission="inventory.read">
                        <InventoryWorkspace canWrite={(authentication.permissions ?? []).includes('inventory.write')} />
                    </PermissionGate>
                    <PermissionGate permission="costing.read">
                        <LaborWorkspace canManage={(authentication.permissions ?? []).includes('costing.manage')} />
                    </PermissionGate>
                    <PermissionGate permission="jobs.read">
                        <JobWorkspace canWrite={(authentication.permissions ?? []).includes('jobs.update')} />
                    </PermissionGate>
                    {feedback && <p className={`feedback feedback--${feedback.tone}`} role="status">{feedback.message}</p>}
                </main>
            </PermissionProvider>
        );
    }

    if (screen === 'login') {
        return (
            <main className="connection-page">
                <section className="connection-card" aria-labelledby="login-title">
                    <Brand branding={branding} />
                    <div className="connection-card__intro">
                        <h1 id="login-title">Entrar na oficina</h1>
                        <p>Use sua conta do servidor configurado em <strong>{baseURL}</strong>.</p>
                    </div>
                    <form onSubmit={submitLogin}>
                        <label htmlFor="login">Usuário ou e-mail</label>
                        <input id="login" autoComplete="username" value={emailOrUsername} disabled={busyAction !== null} onChange={(event) => setEmailOrUsername(event.target.value)} required />
                        <label className="field-label" htmlFor="password">Senha</label>
                        <input id="password" type="password" autoComplete="current-password" value={password} disabled={busyAction !== null} onChange={(event) => setPassword(event.target.value)} required />
                        <p className="connection-card__hint">A senha é enviada somente à camada nativa local e não é armazenada pelo aplicativo.</p>
                        {feedback && <p className={`feedback feedback--${feedback.tone}`} role="status">{feedback.message}</p>}
                        <div className="connection-card__actions">
                            <button className="button button--secondary" type="button" disabled={busyAction !== null} onClick={() => { setFeedback(null); setScreen('connection'); }}>Alterar servidor</button>
                            <button className="button button--primary" type="submit" disabled={busyAction !== null || !emailOrUsername.trim() || !password}>{busyAction === 'login' ? 'Entrando…' : 'Entrar'}</button>
                        </div>
                    </form>
                </section>
            </main>
        );
    }

    return (
        <main className="connection-page">
            <section className="connection-card" aria-labelledby="connection-title">
                <div className="connection-card__intro">
                    <p className="connection-card__eyebrow">Talos · Oficina 3D</p>
                    <h1 id="connection-title">Conectar ao servidor da oficina</h1>
                    <p>Informe o endereço HTTP ou HTTPS do servidor Talos na rede local. Você poderá alterar este endereço antes de entrar.</p>
                </div>
                <form onSubmit={saveConnection}>
                    <label htmlFor="server-base-url">Endereço do servidor</label>
                    <input id="server-base-url" type="url" inputMode="url" autoComplete="url" placeholder="http://oficina.local:8080" value={baseURL} disabled={busyAction !== null} onChange={(event) => { setBaseURL(event.target.value); setConnection(null); setFeedback(null); }} required />
                    <p className="connection-card__hint">Somente o endereço da API é salvo. Credenciais de banco de dados não são aceitas.</p>
                    {feedback && <p className={`feedback feedback--${feedback.tone}`} role="status">{feedback.message}</p>}
                    {connection && <dl className="server-summary"><div><dt>Servidor</dt><dd>{connection.server_version}</dd></div><div><dt>API</dt><dd>{connection.api_version}</dd></div><div><dt>Desktop</dt><dd>{connection.desktop_version}</dd></div></dl>}
                    <div className="connection-card__actions">
                        <button className="button button--secondary" type="button" disabled={busyAction !== null || !baseURL.trim()} onClick={() => void testConnection()}>{busyAction === 'test' ? 'Testando…' : 'Testar conexão'}</button>
                        <button className="button button--primary" type="submit" disabled={busyAction !== null || !baseURL.trim()}>{busyAction === 'save' ? 'Salvando…' : 'Salvar e continuar'}</button>
                    </div>
                </form>
            </section>
        </main>
    );
}

export default App;
