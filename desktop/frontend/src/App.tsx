import {FormEvent, useEffect, useState} from 'react';
import './App.css';
import {
    AuthenticationState,
    ConnectionTestResult,
    getAuthenticationState,
    getServerConnection,
    login,
    logout,
    saveServerConnection,
    testServerConnection,
} from './native';

type Feedback = {tone: 'success' | 'warning' | 'error'; message: string};
type Screen = 'connection' | 'login' | 'shell';

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

function App() {
    const [screen, setScreen] = useState<Screen>('connection');
    const [baseURL, setBaseURL] = useState('');
    const [loading, setLoading] = useState(true);
    const [busyAction, setBusyAction] = useState<'save' | 'test' | 'login' | 'logout' | null>(null);
    const [feedback, setFeedback] = useState<Feedback | null>(null);
    const [connection, setConnection] = useState<ConnectionTestResult | null>(null);
    const [emailOrUsername, setEmailOrUsername] = useState('');
    const [password, setPassword] = useState('');
    const [authentication, setAuthentication] = useState<AuthenticationState | null>(null);

    useEffect(() => {
        void Promise.all([getServerConnection(), getAuthenticationState()])
            .then(([configuration, restored]) => {
                setBaseURL(configuration.server_base_url);
                if (restored.authenticated) {
                    setAuthentication(restored);
                    setScreen('shell');
                } else if (configuration.server_base_url) {
                    setScreen('login');
                }
            })
            .catch((error: unknown) => setFeedback({tone: 'error', message: errorMessage(error)}))
            .finally(() => setLoading(false));
    }, []);

    async function testConnection() {
        setBusyAction('test'); setFeedback(null); setConnection(null);
        try {
            const result = await testServerConnection(baseURL);
            setBaseURL(result.server_base_url); setConnection(result); setFeedback(compatibilityMessage(result));
        } catch (error: unknown) {
            setFeedback({tone: 'error', message: errorMessage(error)});
        } finally { setBusyAction(null); }
    }

    async function saveConnection(event: FormEvent<HTMLFormElement>) {
        event.preventDefault(); setBusyAction('save'); setFeedback(null);
        try {
            const configuration = await saveServerConnection(baseURL);
            setBaseURL(configuration.server_base_url); setConnection(null); setScreen('login');
        } catch (error: unknown) {
            setFeedback({tone: 'error', message: errorMessage(error)});
        } finally { setBusyAction(null); }
    }

    async function submitLogin(event: FormEvent<HTMLFormElement>) {
        event.preventDefault(); setBusyAction('login'); setFeedback(null);
        try {
            const state = await login(emailOrUsername, password);
            setPassword(''); setAuthentication(state); setScreen('shell');
        } catch (error: unknown) {
            setPassword(''); setFeedback({tone: 'error', message: errorMessage(error)});
        } finally { setBusyAction(null); }
    }

    async function submitLogout() {
        setBusyAction('logout'); setFeedback(null);
        try {
            await logout(); setAuthentication(null); setScreen('login');
        } catch (error: unknown) {
            setFeedback({tone: 'error', message: errorMessage(error)});
        } finally { setBusyAction(null); }
    }

    if (loading) return <main className="connection-page"><p>Carregando…</p></main>;

    if (screen === 'shell' && authentication) {
        return (
            <main className="app-shell">
                <header className="app-shell__header">
                    <div><p className="connection-card__eyebrow">Talos · Oficina 3D</p><strong>Área de trabalho</strong></div>
                    <div className="app-shell__account"><span>{authentication.user_name}</span><button className="button button--secondary" disabled={busyAction === 'logout'} onClick={() => void submitLogout()}>{busyAction === 'logout' ? 'Saindo…' : 'Sair'}</button></div>
                </header>
                <section className="app-shell__empty"><h1>Sessão iniciada</h1><p>A navegação da oficina será adicionada nos próximos pacotes.</p>{feedback && <p className={`feedback feedback--${feedback.tone}`} role="status">{feedback.message}</p>}</section>
            </main>
        );
    }

    if (screen === 'login') {
        return (
            <main className="connection-page">
                <section className="connection-card" aria-labelledby="login-title">
                    <div className="connection-card__intro"><p className="connection-card__eyebrow">Talos · Oficina 3D</p><h1 id="login-title">Entrar na oficina</h1><p>Use sua conta do servidor configurado em <strong>{baseURL}</strong>.</p></div>
                    <form onSubmit={submitLogin}>
                        <label htmlFor="login">Usuário ou e-mail</label>
                        <input id="login" autoComplete="username" value={emailOrUsername} disabled={busyAction !== null} onChange={(event) => setEmailOrUsername(event.target.value)} required />
                        <label className="field-label" htmlFor="password">Senha</label>
                        <input id="password" type="password" autoComplete="current-password" value={password} disabled={busyAction !== null} onChange={(event) => setPassword(event.target.value)} required />
                        <p className="connection-card__hint">A senha é enviada somente à camada nativa local e não é armazenada pelo aplicativo.</p>
                        {feedback && <p className={`feedback feedback--${feedback.tone}`} role="status">{feedback.message}</p>}
                        <div className="connection-card__actions"><button className="button button--secondary" type="button" disabled={busyAction !== null} onClick={() => {setFeedback(null); setScreen('connection');}}>Alterar servidor</button><button className="button button--primary" type="submit" disabled={busyAction !== null || !emailOrUsername.trim() || !password}>{busyAction === 'login' ? 'Entrando…' : 'Entrar'}</button></div>
                    </form>
                </section>
            </main>
        );
    }

    return (
        <main className="connection-page">
            <section className="connection-card" aria-labelledby="connection-title">
                <div className="connection-card__intro"><p className="connection-card__eyebrow">Talos · Oficina 3D</p><h1 id="connection-title">Conectar ao servidor da oficina</h1><p>Informe o endereço HTTP ou HTTPS do servidor Talos na rede local. Você poderá alterar este endereço antes de entrar.</p></div>
                <form onSubmit={saveConnection}>
                    <label htmlFor="server-base-url">Endereço do servidor</label>
                    <input id="server-base-url" type="url" inputMode="url" autoComplete="url" placeholder="http://oficina.local:8080" value={baseURL} disabled={busyAction !== null} onChange={(event) => {setBaseURL(event.target.value); setConnection(null); setFeedback(null);}} required />
                    <p className="connection-card__hint">Somente o endereço da API é salvo. Credenciais de banco de dados não são aceitas.</p>
                    {feedback && <p className={`feedback feedback--${feedback.tone}`} role="status">{feedback.message}</p>}
                    {connection && <dl className="server-summary"><div><dt>Servidor</dt><dd>{connection.server_version}</dd></div><div><dt>API</dt><dd>{connection.api_version}</dd></div><div><dt>Desktop</dt><dd>{connection.desktop_version}</dd></div></dl>}
                    <div className="connection-card__actions"><button className="button button--secondary" type="button" disabled={busyAction !== null || !baseURL.trim()} onClick={() => void testConnection()}>{busyAction === 'test' ? 'Testando…' : 'Testar conexão'}</button><button className="button button--primary" type="submit" disabled={busyAction !== null || !baseURL.trim()}>{busyAction === 'save' ? 'Salvando…' : 'Salvar e continuar'}</button></div>
                </form>
            </section>
        </main>
    );
}

export default App;
