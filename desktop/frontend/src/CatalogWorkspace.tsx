import {FormEvent, useEffect, useMemo, useState} from 'react';
import {
    CatalogItem,
    CatalogItemInput,
    CatalogPurpose,
    CatalogStatus,
    createCatalogItem,
    listCatalogItems,
    updateCatalogItem,
} from './native';

const purposeLabels: Record<CatalogPurpose, string> = {
    product: 'Produto',
    prototype: 'Protótipo',
    tooling: 'Ferramental',
    test: 'Teste',
    internal: 'Interno',
    personal: 'Pessoal',
};

const statusLabels: Record<CatalogStatus, string> = {
    active: 'Ativo',
    archived: 'Arquivado',
};

type Draft = {
    name: string;
    sku: string;
    description: string;
    purpose: CatalogPurpose;
    sellable: boolean;
    tags: string;
    status: CatalogStatus;
};

const emptyDraft: Draft = {
    name: '', sku: '', description: '', purpose: 'product', sellable: false, tags: '', status: 'active',
};

function draftFrom(item: CatalogItem): Draft {
    return {
        name: item.name,
        sku: item.sku ?? '',
        description: item.description,
        purpose: item.purpose,
        sellable: item.sellable,
        tags: item.tags.join(', '),
        status: item.status,
    };
}

function messageFrom(error: unknown): string {
    if (error instanceof Error) return error.message;
    return typeof error === 'string' ? error : 'Não foi possível concluir a operação.';
}

export function CatalogWorkspace({canWrite}: {canWrite: boolean}) {
    const [items, setItems] = useState<CatalogItem[]>([]);
    const [total, setTotal] = useState(0);
    const [selectedID, setSelectedID] = useState<string | null>(null);
    const [draft, setDraft] = useState<Draft>(emptyDraft);
    const [editing, setEditing] = useState(false);
    const [loading, setLoading] = useState(true);
    const [saving, setSaving] = useState(false);
    const [feedback, setFeedback] = useState<string | null>(null);

    const selected = useMemo(() => items.find((item) => item.id === selectedID) ?? null, [items, selectedID]);

    useEffect(() => {
        let active = true;
        void listCatalogItems()
            .then((page) => {
                if (!active) return;
                setItems(page.items);
                setTotal(page.pagination.total);
                setSelectedID(page.items[0]?.id ?? null);
            })
            .catch((error: unknown) => active && setFeedback(messageFrom(error)))
            .finally(() => active && setLoading(false));
        return () => { active = false; };
    }, []);

    function beginCreate() {
        setSelectedID(null);
        setDraft(emptyDraft);
        setFeedback(null);
        setEditing(true);
    }

    function beginEdit() {
        if (!selected) return;
        setDraft(draftFrom(selected));
        setFeedback(null);
        setEditing(true);
    }

    function cancelEdit() {
        setEditing(false);
        setFeedback(null);
        if (!selectedID && items.length > 0) setSelectedID(items[0].id);
    }

    async function save(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setSaving(true);
        setFeedback(null);
        const input: CatalogItemInput = {
            name: draft.name,
            sku: draft.sku.trim() || null,
            description: draft.description,
            purpose: draft.purpose,
            sellable: draft.sellable,
            tags: draft.tags.split(',').map((tag) => tag.trim()).filter(Boolean),
            status: draft.status,
        };
        try {
            const saved = selected
                ? await updateCatalogItem(selected.id, input)
                : await createCatalogItem(input);
            setItems((current) => {
                const withoutSaved = current.filter((item) => item.id !== saved.id);
                return [...withoutSaved, saved].sort((left, right) => left.name.localeCompare(right.name));
            });
            if (!selected) setTotal((current) => current + 1);
            setSelectedID(saved.id);
            setEditing(false);
        } catch (error: unknown) {
            setFeedback(messageFrom(error));
        } finally {
            setSaving(false);
        }
    }

    return (
        <section className="catalog-workspace" aria-labelledby="catalog-title">
            <div className="catalog-workspace__toolbar">
                <div>
                    <p className="connection-card__eyebrow">Catálogo</p>
                    <h1 id="catalog-title">Itens da oficina</h1>
                    <p>{total} {total === 1 ? 'item' : 'itens'} no catálogo.</p>
                </div>
                {canWrite && <button className="button button--primary" type="button" onClick={beginCreate}>Novo item</button>}
            </div>

            {feedback && <p className="feedback feedback--error" role="alert">{feedback}</p>}
            {loading ? <p className="catalog-workspace__loading">Carregando catálogo…</p> : (
                <div className="catalog-layout">
                    <nav className="catalog-list" aria-label="Itens do catálogo">
                        {items.length === 0 && <p>Nenhum item cadastrado.</p>}
                        {items.map((item) => (
                            <button
                                className={`catalog-list__item${item.id === selectedID ? ' catalog-list__item--selected' : ''}`}
                                key={item.id}
                                type="button"
                                onClick={() => { setSelectedID(item.id); setEditing(false); setFeedback(null); }}
                            >
                                <strong>{item.name}</strong>
                                <span>{purposeLabels[item.purpose]} · {statusLabels[item.status]}</span>
                                {item.sku && <small>{item.sku}</small>}
                            </button>
                        ))}
                    </nav>

                    <article className="catalog-detail">
                        {editing ? (
                            <CatalogForm draft={draft} setDraft={setDraft} saving={saving} isUpdate={selected !== null} onSubmit={save} onCancel={cancelEdit} />
                        ) : selected ? (
                            <>
                                <div className="catalog-detail__heading">
                                    <div><span className="catalog-status">{statusLabels[selected.status]}</span><h2>{selected.name}</h2></div>
                                    {canWrite && <button className="button button--secondary" type="button" onClick={beginEdit}>Editar</button>}
                                </div>
                                <dl className="catalog-detail__facts">
                                    <div><dt>Finalidade</dt><dd>{purposeLabels[selected.purpose]}</dd></div>
                                    <div><dt>Vendável</dt><dd>{selected.sellable ? 'Sim' : 'Não'}</dd></div>
                                    <div><dt>SKU</dt><dd>{selected.sku || '—'}</dd></div>
                                </dl>
                                <p>{selected.description || 'Sem descrição.'}</p>
                                <div className="catalog-tags">{selected.tags.map((tag) => <span key={tag}>{tag}</span>)}</div>
                            </>
                        ) : <p>Selecione um item ou crie o primeiro cadastro.</p>}
                    </article>
                </div>
            )}
        </section>
    );
}

function CatalogForm({draft, setDraft, saving, isUpdate, onSubmit, onCancel}: {
    draft: Draft;
    setDraft: (draft: Draft) => void;
    saving: boolean;
    isUpdate: boolean;
    onSubmit: (event: FormEvent<HTMLFormElement>) => void;
    onCancel: () => void;
}) {
    return (
        <form className="catalog-form" onSubmit={onSubmit}>
            <h2>{isUpdate ? 'Editar item' : 'Novo item'}</h2>
            <label>Nome<input value={draft.name} maxLength={200} required disabled={saving} onChange={(event) => setDraft({...draft, name: event.target.value})} /></label>
            <label>SKU opcional<input value={draft.sku} maxLength={100} disabled={saving} onChange={(event) => setDraft({...draft, sku: event.target.value})} /></label>
            <label>Descrição<textarea value={draft.description} maxLength={10000} disabled={saving} onChange={(event) => setDraft({...draft, description: event.target.value})} /></label>
            <div className="catalog-form__row">
                <label>Finalidade<select value={draft.purpose} disabled={saving} onChange={(event) => setDraft({...draft, purpose: event.target.value as CatalogPurpose})}>{Object.entries(purposeLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
                <label>Status<select value={draft.status} disabled={saving} onChange={(event) => setDraft({...draft, status: event.target.value as CatalogStatus})}>{Object.entries(statusLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
            </div>
            <label>Tags separadas por vírgula<input value={draft.tags} disabled={saving} onChange={(event) => setDraft({...draft, tags: event.target.value})} /></label>
            <label className="catalog-form__checkbox"><input type="checkbox" checked={draft.sellable} disabled={saving} onChange={(event) => setDraft({...draft, sellable: event.target.checked})} />Item vendável</label>
            <div className="connection-card__actions">
                <button className="button button--secondary" type="button" disabled={saving} onClick={onCancel}>Cancelar</button>
                <button className="button button--primary" type="submit" disabled={saving}>{saving ? 'Salvando…' : 'Salvar item'}</button>
            </div>
        </form>
    );
}
