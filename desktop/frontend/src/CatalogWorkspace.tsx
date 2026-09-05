import {FormEvent, useEffect, useMemo, useState} from 'react';
import {
    CatalogItem,
    CatalogItemInput,
    CatalogPurpose,
    CatalogStatus,
    CatalogPart,
    CatalogBOMItemInput,
    CatalogBOMPreview,
    Supply,
    DesignFileRole,
    DesignOrigin,
    DesignVersion,
    DesignVersionInput,
    attachDesignFile,
    createCatalogPart,
    createCatalogItem,
    createCatalogBOMItem,
    deleteCatalogBOMItem,
    getCatalogBOM,
    createDesignVersion,
    listCatalogParts,
    listCatalogItems,
    listDesignVersions,
    listSupplies,
    updateCatalogItem,
    updateCatalogBOMItem,
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
                                <SupplyBOM key={`bom-${selected.id}`} item={selected} canWrite={canWrite} onError={setFeedback} />
                                <DesignHistory key={selected.id} item={selected} canWrite={canWrite} onError={setFeedback} />
                            </>
                        ) : <p>Selecione um item ou crie o primeiro cadastro.</p>}
                    </article>
                </div>
            )}
        </section>
    );
}

const emptyBOMInput: CatalogBOMItemInput = {supply_id: '', quantity_per_unit: '1', waste_percent: '0', notes: ''};

function SupplyBOM({item, canWrite, onError}: {item: CatalogItem; canWrite: boolean; onError: (message: string | null) => void}) {
    const [preview, setPreview] = useState<CatalogBOMPreview | null>(null);
    const [supplies, setSupplies] = useState<Supply[]>([]);
    const [draft, setDraft] = useState<CatalogBOMItemInput>(emptyBOMInput);
    const [editingID, setEditingID] = useState<string | null>(null);
    const [busy, setBusy] = useState(false);

    async function reload() {
        setPreview(await getCatalogBOM(item.id));
    }

    useEffect(() => {
        let active = true;
        void getCatalogBOM(item.id)
            .then((value) => active && setPreview(value))
            .catch((error: unknown) => active && onError(messageFrom(error)));
        if (canWrite) {
            void listSupplies().then((values) => active && setSupplies(values)).catch(() => undefined);
        }
        return () => { active = false; };
    }, [item.id, canWrite, onError]);

    async function save(event: FormEvent<HTMLFormElement>) {
        event.preventDefault(); setBusy(true); onError(null);
        try {
            if (editingID) await updateCatalogBOMItem(item.id, editingID, draft);
            else await createCatalogBOMItem(item.id, draft);
            await reload();
            setDraft(emptyBOMInput); setEditingID(null);
        } catch (error: unknown) { onError(messageFrom(error)); } finally { setBusy(false); }
    }

    async function remove(id: string) {
        setBusy(true); onError(null);
        try {
            await deleteCatalogBOMItem(item.id, id);
            await reload();
            if (editingID === id) { setEditingID(null); setDraft(emptyBOMInput); }
        } catch (error: unknown) { onError(messageFrom(error)); } finally { setBusy(false); }
    }

    return <section className="supply-bom" aria-labelledby="supply-bom-title">
        <div className="design-history__heading"><div><h3 id="supply-bom-title">Insumos por unidade</h3><p>Quantidades e perdas previstas para produzir uma unidade deste item.</p></div></div>
        {!preview && <p>Carregando insumos…</p>}
        {preview && preview.items.length === 0 && <p>Nenhum insumo vinculado.</p>}
        {preview && preview.items.length > 0 && <div className="supply-bom__table">
            {preview.items.map((line) => <article key={line.id} className="supply-bom__line">
                <div><strong>{line.supply_name}</strong><span>{line.quantity_per_unit} {line.supply_unit} + {line.waste_percent}% de perda</span></div>
                <div><span>{line.effective_quantity_per_unit} {line.supply_unit} efetivos</span><strong>{line.exact_replacement_cost_cents_per_unit} centavos</strong></div>
                {canWrite && <div className="supply-bom__actions">
                    <button className="button button--secondary" type="button" disabled={busy} onClick={() => { setEditingID(line.id); setDraft({supply_id: line.supply_id, quantity_per_unit: line.quantity_per_unit, waste_percent: line.waste_percent, notes: line.notes}); }}>Editar</button>
                    <button className="button button--secondary" type="button" disabled={busy} onClick={() => void remove(line.id)}>Remover</button>
                </div>}
            </article>)}
            <p className="supply-bom__total"><strong>Total exato:</strong> {preview.exact_total_replacement_cost_cents} centavos <small>(sem arredondamento)</small></p>
        </div>}
        {canWrite && <form className="design-inline-form supply-bom__form" onSubmit={save}>
            <label>Insumo
                <input required list={`supply-options-${item.id}`} value={draft.supply_id} disabled={busy} placeholder="ID do insumo" onChange={(event) => setDraft({...draft, supply_id: event.target.value})} />
                <datalist id={`supply-options-${item.id}`}>{supplies.map((supply) => <option key={supply.id} value={supply.id}>{supply.name} ({supply.unit})</option>)}</datalist>
            </label>
            <label>Quantidade por unidade<input required type="number" min="0.000001" step="0.000001" value={draft.quantity_per_unit} disabled={busy} onChange={(event) => setDraft({...draft, quantity_per_unit: event.target.value})} /></label>
            <label>Perda (%)<input required type="number" min="0" step="0.0001" value={draft.waste_percent} disabled={busy} onChange={(event) => setDraft({...draft, waste_percent: event.target.value})} /></label>
            <label>Notas<input maxLength={10000} value={draft.notes} disabled={busy} onChange={(event) => setDraft({...draft, notes: event.target.value})} /></label>
            <button className="button button--primary" disabled={busy} type="submit">{editingID ? 'Salvar insumo' : 'Adicionar insumo'}</button>
            {editingID && <button className="button button--secondary" disabled={busy} type="button" onClick={() => { setEditingID(null); setDraft(emptyBOMInput); }}>Cancelar</button>}
        </form>}
    </section>;
}

const originLabels: Record<DesignOrigin, string> = {
    original: 'Original', customer: 'Cliente', remix: 'Remix', third_party: 'Terceiro', unknown: 'Desconhecida',
};

const fileRoleLabels: Record<DesignFileRole, string> = {
    source: 'Fonte', mesh: 'Malha', print: 'Impressão', preview: 'Prévia', documentation: 'Documentação', other: 'Outro',
};

type VersionDraft = {
    version: string;
    notes: string;
    origin: DesignOrigin;
    sourceURL: string;
    originalAuthor: string;
    licenseName: string;
    commercialPermission: 'unknown' | 'allowed' | 'denied';
    attributionRequired: boolean;
    attributionText: string;
};

const emptyVersionDraft: VersionDraft = {
    version: '', notes: '', origin: 'unknown', sourceURL: '', originalAuthor: '', licenseName: '',
    commercialPermission: 'unknown', attributionRequired: false, attributionText: '',
};

function DesignHistory({item, canWrite, onError}: {item: CatalogItem; canWrite: boolean; onError: (message: string | null) => void}) {
    const [parts, setParts] = useState<CatalogPart[]>([]);
    const [selectedPartID, setSelectedPartID] = useState<string | null>(null);
    const [versionsByPart, setVersionsByPart] = useState<Record<string, DesignVersion[]>>({});
    const [partName, setPartName] = useState('');
    const [partQuantity, setPartQuantity] = useState(1);
    const [versionDraft, setVersionDraft] = useState<VersionDraft>(emptyVersionDraft);
    const [fileID, setFileID] = useState('');
    const [fileRole, setFileRole] = useState<DesignFileRole>('print');
    const [busy, setBusy] = useState(false);

    useEffect(() => {
        let active = true;
        void listCatalogParts(item.id).then(async (loadedParts) => {
            const histories = await Promise.all(loadedParts.map(async (part) => [part.id, await listDesignVersions(part.id)] as const));
            if (!active) return;
            setParts(loadedParts);
            setSelectedPartID(loadedParts[0]?.id ?? null);
            setVersionsByPart(Object.fromEntries(histories));
        }).catch((error: unknown) => active && onError(messageFrom(error)));
        return () => { active = false; };
    }, [item.id, onError]);

    const selectedPart = parts.find((part) => part.id === selectedPartID) ?? null;
    const versions = selectedPartID ? versionsByPart[selectedPartID] ?? [] : [];
    const latestVersions = parts.map((part) => versionsByPart[part.id]?.[0]);
    const hasDenied = latestVersions.some((version) => version?.commercial_use_allowed === false);
    const hasUnknown = parts.length === 0 || latestVersions.some((version) => !version || version.commercial_use_allowed == null);

    async function addPart(event: FormEvent<HTMLFormElement>) {
        event.preventDefault(); setBusy(true); onError(null);
        try {
            const part = await createCatalogPart(item.id, {name: partName, quantity: partQuantity, notes: ''});
            setParts((current) => [...current, part].sort((left, right) => left.name.localeCompare(right.name)));
            setVersionsByPart((current) => ({...current, [part.id]: []}));
            setSelectedPartID(part.id); setPartName(''); setPartQuantity(1);
        } catch (error: unknown) { onError(messageFrom(error)); } finally { setBusy(false); }
    }

    async function addVersion(event: FormEvent<HTMLFormElement>) {
        event.preventDefault(); if (!selectedPart) return; setBusy(true); onError(null);
        const commercialUseAllowed = versionDraft.commercialPermission === 'unknown' ? null : versionDraft.commercialPermission === 'allowed';
        const input: DesignVersionInput = {
            version: versionDraft.version, notes: versionDraft.notes, origin: versionDraft.origin,
            source_url: versionDraft.sourceURL.trim() || null, original_author: versionDraft.originalAuthor,
            license_name: versionDraft.licenseName, commercial_use_allowed: commercialUseAllowed,
            attribution_required: versionDraft.attributionRequired, attribution_text: versionDraft.attributionText,
        };
        try {
            const created = await createDesignVersion(selectedPart.id, input);
            setVersionsByPart((current) => ({...current, [selectedPart.id]: [created, ...(current[selectedPart.id] ?? [])]}));
            setVersionDraft(emptyVersionDraft);
        } catch (error: unknown) { onError(messageFrom(error)); } finally { setBusy(false); }
    }

    async function linkFile(event: FormEvent<HTMLFormElement>) {
        event.preventDefault(); const latest = versions[0]; if (!latest) return; setBusy(true); onError(null);
        try {
            const file = await attachDesignFile(latest.id, fileID, fileRole);
            setVersionsByPart((current) => ({...current, [latest.catalog_part_id]: (current[latest.catalog_part_id] ?? []).map((version) => version.id === latest.id ? {...version, files: [...version.files, file]} : version)}));
            setFileID('');
        } catch (error: unknown) { onError(messageFrom(error)); } finally { setBusy(false); }
    }

    return <section className="design-history" aria-labelledby="design-history-title">
        <div className="design-history__heading"><div><h3 id="design-history-title">Partes e designs</h3><p>Versões preservam a procedência e os arquivos usados.</p></div></div>
        {item.sellable && hasDenied && <p className="feedback feedback--error" role="status"><strong>Venda não autorizada.</strong> A versão atual de ao menos uma parte nega uso comercial. O aviso não bloqueia uso interno ou protótipos.</p>}
        {item.sellable && !hasDenied && hasUnknown && <p className="feedback feedback--warning" role="status"><strong>Permissão comercial desconhecida.</strong> Cadastre a licença de todas as partes antes de vender. Uso interno ou de protótipo continua permitido.</p>}
        <div className="design-history__parts">
            {parts.map((part) => <button key={part.id} type="button" className={part.id === selectedPartID ? 'design-tab design-tab--selected' : 'design-tab'} onClick={() => setSelectedPartID(part.id)}>{part.name} <small>×{part.quantity}</small></button>)}
        </div>
        {canWrite && <form className="design-inline-form" onSubmit={addPart}>
            <label>Nova parte<input required maxLength={200} value={partName} disabled={busy} onChange={(event) => setPartName(event.target.value)} /></label>
            <label>Quantidade<input required min={1} type="number" value={partQuantity} disabled={busy} onChange={(event) => setPartQuantity(Number(event.target.value))} /></label>
            <button className="button button--secondary" disabled={busy} type="submit">Adicionar parte</button>
        </form>}
        {selectedPart && <div className="design-versions">
            <h4>Histórico de {selectedPart.name}</h4>
            {versions.length === 0 && <p>Nenhuma versão cadastrada.</p>}
            {versions.map((version) => <article className="design-version" key={version.id}>
                <div><strong>{version.version}</strong><span>{originLabels[version.origin]}</span></div>
                <p>{version.license_name || 'Licença não informada'} · {version.commercial_use_allowed == null ? 'uso comercial desconhecido' : version.commercial_use_allowed ? 'uso comercial permitido' : 'uso comercial negado'}</p>
                {version.source_url && <a href={version.source_url} target="_blank" rel="noreferrer">Abrir origem</a>}
                <ul>{version.files.map((file) => <li key={`${file.file_id}-${file.role}`}><span>{fileRoleLabels[file.role]}</span> {file.original_name}{file.role === 'print' && <strong> · arquivo de impressão</strong>}</li>)}</ul>
            </article>)}
            {canWrite && <>
                <form className="catalog-form design-version-form" onSubmit={addVersion}>
                    <h4>Nova versão imutável</h4>
                    <div className="catalog-form__row"><label>Versão<input required maxLength={100} value={versionDraft.version} disabled={busy} onChange={(event) => setVersionDraft({...versionDraft, version: event.target.value})} /></label><label>Origem<select value={versionDraft.origin} disabled={busy} onChange={(event) => setVersionDraft({...versionDraft, origin: event.target.value as DesignOrigin})}>{Object.entries(originLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label></div>
                    <label>URL de origem<input type="url" value={versionDraft.sourceURL} disabled={busy} onChange={(event) => setVersionDraft({...versionDraft, sourceURL: event.target.value})} /></label>
                    <div className="catalog-form__row"><label>Autor original<input maxLength={200} value={versionDraft.originalAuthor} disabled={busy} onChange={(event) => setVersionDraft({...versionDraft, originalAuthor: event.target.value})} /></label><label>Licença<input maxLength={200} value={versionDraft.licenseName} disabled={busy} onChange={(event) => setVersionDraft({...versionDraft, licenseName: event.target.value})} /></label></div>
                    <label>Uso comercial<select value={versionDraft.commercialPermission} disabled={busy} onChange={(event) => setVersionDraft({...versionDraft, commercialPermission: event.target.value as VersionDraft['commercialPermission']})}><option value="unknown">Desconhecido</option><option value="allowed">Permitido</option><option value="denied">Negado</option></select></label>
                    <label className="catalog-form__checkbox"><input type="checkbox" checked={versionDraft.attributionRequired} disabled={busy} onChange={(event) => setVersionDraft({...versionDraft, attributionRequired: event.target.checked})} />Exige atribuição</label>
                    {versionDraft.attributionRequired && <label>Texto de atribuição<input required maxLength={4000} value={versionDraft.attributionText} disabled={busy} onChange={(event) => setVersionDraft({...versionDraft, attributionText: event.target.value})} /></label>}
                    <label>Notas<textarea maxLength={10000} value={versionDraft.notes} disabled={busy} onChange={(event) => setVersionDraft({...versionDraft, notes: event.target.value})} /></label>
                    <button className="button button--primary" disabled={busy} type="submit">Criar versão</button>
                </form>
                {versions[0] && <form className="design-inline-form" onSubmit={linkFile}>
                    <label>ID de arquivo existente<input required value={fileID} disabled={busy} onChange={(event) => setFileID(event.target.value)} /></label>
                    <label>Função<select value={fileRole} disabled={busy} onChange={(event) => setFileRole(event.target.value as DesignFileRole)}>{Object.entries(fileRoleLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
                    <button className="button button--secondary" disabled={busy} type="submit">Vincular ao design atual</button>
                </form>}
            </>}
        </div>}
    </section>;
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
