import {FormEvent, useEffect, useMemo, useState} from 'react';
import {
    Material,
    Spool,
    SpoolMeasurement,
    listMaterials,
    listSpoolMeasurements,
    listSpools,
    recordSpoolMeasurement,
} from './native';

function errorMessage(error: unknown): string {
    if (error instanceof Error) return error.message;
    return typeof error === 'string' ? error : 'Não foi possível concluir a operação.';
}

export function InventoryWorkspace({canWrite}: {canWrite: boolean}) {
    const [materials, setMaterials] = useState<Material[]>([]);
    const [spools, setSpools] = useState<Spool[]>([]);
    const [selectedID, setSelectedID] = useState<string | null>(null);
    const [measurements, setMeasurements] = useState<SpoolMeasurement[]>([]);
    const [gross, setGross] = useState('');
    const [notes, setNotes] = useState('');
    const [busy, setBusy] = useState(false);
    const [feedback, setFeedback] = useState<string | null>(null);

    const selected = useMemo(
        () => spools.find((spool) => spool.id === selectedID) ?? null,
        [spools, selectedID],
    );
    const materialNames = useMemo(
        () => new Map(materials.map((material) => [material.id, `${material.manufacturer} ${material.name}`])),
        [materials],
    );

    useEffect(() => {
        let active = true;
        void Promise.all([listMaterials(), listSpools()])
            .then(([loadedMaterials, loadedSpools]) => {
                if (!active) return;
                setMaterials(loadedMaterials);
                setSpools(loadedSpools);
                setSelectedID(loadedSpools[0]?.id ?? null);
            })
            .catch((error: unknown) => {
                if (active) setFeedback(errorMessage(error));
            });
        return () => {
            active = false;
        };
    }, []);

    useEffect(() => {
        if (!selectedID) return;
        let active = true;
        void listSpoolMeasurements(selectedID)
            .then((values) => {
                if (active) setMeasurements(values);
            })
            .catch((error: unknown) => {
                if (active) setFeedback(errorMessage(error));
            });
        return () => {
            active = false;
        };
    }, [selectedID]);

    async function weigh(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        if (!selected) return;
        setBusy(true);
        setFeedback(null);
        try {
            const value = await recordSpoolMeasurement(selected.id, {
                measured_at: new Date().toISOString(),
                gross_weight_g: gross,
                source: 'manual',
                notes,
            });
            setMeasurements((current) => [value, ...current]);
            setSpools((current) => current.map((spool) => spool.id === selected.id
                ? {
                    ...spool,
                    current_remaining_weight_g: value.derived_remaining_weight_g,
                    last_weighed_at: value.measured_at,
                }
                : spool));
            setGross('');
            setNotes('');
        } catch (error: unknown) {
            setFeedback(errorMessage(error));
        } finally {
            setBusy(false);
        }
    }

    return (
        <section className="inventory-workspace" aria-labelledby="inventory-title">
            <div className="catalog-workspace__toolbar">
                <div>
                    <p className="connection-card__eyebrow">Inventário</p>
                    <h1 id="inventory-title">Bobinas e pesagens</h1>
                    <p>{spools.length} {spools.length === 1 ? 'bobina cadastrada' : 'bobinas cadastradas'}.</p>
                </div>
            </div>
            {feedback && <p className="feedback feedback--error" role="alert">{feedback}</p>}
            <div className="inventory-layout">
                <nav className="catalog-list" aria-label="Bobinas">
                    {spools.length === 0 && <p>Nenhuma bobina cadastrada.</p>}
                    {spools.map((spool) => (
                        <button
                            className={`catalog-list__item${spool.id === selectedID ? ' catalog-list__item--selected' : ''}`}
                            key={spool.id}
                            type="button"
                            onClick={() => setSelectedID(spool.id)}
                        >
                            <strong>{spool.code}</strong>
                            <span>{materialNames.get(spool.material_id) ?? 'Material desconhecido'} · {spool.status}</span>
                            <small>{spool.current_remaining_weight_g ?? '—'} g restantes</small>
                        </button>
                    ))}
                </nav>
                <article className="catalog-detail">
                    {selected ? (
                        <>
                            <div className="catalog-detail__heading">
                                <div>
                                    <span className="catalog-status">{selected.status}</span>
                                    <h2>{selected.code}</h2>
                                </div>
                            </div>
                            <dl className="catalog-detail__facts">
                                <div><dt>Material</dt><dd>{materialNames.get(selected.material_id) ?? '—'}</dd></div>
                                <div><dt>Tara</dt><dd>{selected.tare_weight_g} g</dd></div>
                                <div>
                                    <dt>Saldo atual</dt>
                                    <dd>{selected.current_remaining_weight_g ?? 'Sem pesagem'}{selected.current_remaining_weight_g ? ' g' : ''}</dd>
                                </div>
                            </dl>
                            {canWrite && (
                                <form className="design-inline-form" onSubmit={weigh}>
                                    <label>
                                        Peso bruto (g)
                                        <input required inputMode="decimal" pattern="[0-9]+([.][0-9]{1,3})?" value={gross} disabled={busy} onChange={(event) => setGross(event.target.value)} />
                                    </label>
                                    <label>
                                        Notas
                                        <input value={notes} maxLength={10000} disabled={busy} onChange={(event) => setNotes(event.target.value)} />
                                    </label>
                                    <button className="button button--primary" disabled={busy} type="submit">Registrar pesagem</button>
                                </form>
                            )}
                            <h3>Histórico imutável</h3>
                            {measurements.length === 0 ? (
                                <p>Nenhuma pesagem registrada.</p>
                            ) : (
                                <ol className="measurement-history">
                                    {measurements.map((measurement) => (
                                        <li key={measurement.id}>
                                            <strong>{measurement.derived_remaining_weight_g} g restantes</strong>
                                            <span>Bruto {measurement.gross_weight_g} g · {new Date(measurement.measured_at).toLocaleString('pt-BR')}</span>
                                            {measurement.notes && <small>{measurement.notes}</small>}
                                        </li>
                                    ))}
                                </ol>
                            )}
                        </>
                    ) : <p>Selecione uma bobina.</p>}
                </article>
            </div>
        </section>
    );
}
