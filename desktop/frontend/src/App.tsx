import {FormEvent, useEffect, useState} from 'react';
import './App.css';
import {
    ConnectionTestResult,
    getServerConnection,
    saveServerConnection,
    testServerConnection,
} from './native';

type Feedback = {
    tone: 'success' | 'warning' | 'error';
    message: string;
};

function errorMessage(error: unknown): string {
    if (error instanceof Error) {
        return error.message;
    }
    return typeof error === 'string' ? error : 'Não foi possível concluir a operação.';
}

function compatibilityMessage(result: ConnectionTestResult): Feedback {
    if (result.compatible) {
        return {tone: 'success', message: `Conectado a ${result.workshop_name}.`};
    }
    if (result.compatibility_issue === 'desktop_update_required') {
        return {
            tone: 'warning',
            message: `Este servidor requer o desktop ${result.minimum_desktop_version} ou mais recente.`,
        };
    }
    return {tone: 'warning', message: `A API ${result.api_version} não é compatível com este aplicativo.`};
}

function App() {
    const [baseURL, setBaseURL] = useState('');
    const [loading, setLoading] = useState(true);
    const [busyAction, setBusyAction] = useState<'save' | 'test' | null>(null);
    const [feedback, setFeedback] = useState<Feedback | null>(null);
    const [connection, setConnection] = useState<ConnectionTestResult | null>(null);

    useEffect(() => {
        void getServerConnection()
            .then((configuration) => setBaseURL(configuration.server_base_url))
            .catch((error: unknown) => setFeedback({tone: 'error', message: errorMessage(error)}))
            .finally(() => setLoading(false));
    }, []);

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
            setFeedback({tone: 'success', message: 'Endereço salvo neste computador.'});
        } catch (error: unknown) {
            setFeedback({tone: 'error', message: errorMessage(error)});
        } finally {
            setBusyAction(null);
        }
    }

    return (
        <main className="connection-page">
            <section className="connection-card" aria-labelledby="connection-title">
                <div className="connection-card__intro">
                    <p className="connection-card__eyebrow">Talos · Oficina 3D</p>
                    <h1 id="connection-title">Conectar ao servidor da oficina</h1>
                    <p>
                        Informe o endereço HTTP ou HTTPS do servidor Talos na rede local.
                        Você poderá alterar este endereço antes de entrar.
                    </p>
                </div>

                <form onSubmit={saveConnection}>
                    <label htmlFor="server-base-url">Endereço do servidor</label>
                    <input
                        id="server-base-url"
                        name="server-base-url"
                        type="url"
                        inputMode="url"
                        autoComplete="url"
                        placeholder="http://oficina.local:8080"
                        value={baseURL}
                        disabled={loading || busyAction !== null}
                        onChange={(event) => {
                            setBaseURL(event.target.value);
                            setConnection(null);
                            setFeedback(null);
                        }}
                        required
                    />
                    <p className="connection-card__hint">
                        Somente o endereço da API é salvo. Credenciais de banco de dados não são aceitas.
                    </p>

                    {feedback && (
                        <p className={`feedback feedback--${feedback.tone}`} role="status">
                            {feedback.message}
                        </p>
                    )}

                    {connection && (
                        <dl className="server-summary">
                            <div><dt>Servidor</dt><dd>{connection.server_version}</dd></div>
                            <div><dt>API</dt><dd>{connection.api_version}</dd></div>
                            <div><dt>Desktop</dt><dd>{connection.desktop_version}</dd></div>
                        </dl>
                    )}

                    <div className="connection-card__actions">
                        <button
                            className="button button--secondary"
                            type="button"
                            disabled={loading || busyAction !== null || baseURL.trim() === ''}
                            onClick={() => void testConnection()}
                        >
                            {busyAction === 'test' ? 'Testando…' : 'Testar conexão'}
                        </button>
                        <button
                            className="button button--primary"
                            type="submit"
                            disabled={loading || busyAction !== null || baseURL.trim() === ''}
                        >
                            {busyAction === 'save' ? 'Salvando…' : 'Salvar endereço'}
                        </button>
                    </div>
                </form>
            </section>
        </main>
    );
}

export default App;
