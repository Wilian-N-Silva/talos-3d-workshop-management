import {FormEvent, useEffect, useMemo, useState} from 'react';
import {
    LowInventory,
    Material,
    Spool,
    SpoolMeasurement,
    Supply,
    SupplyMovement,
    SupplyMovementType,
    createSupply,
    listLowInventory,
    listMaterials,
    listSpoolMeasurements,
    listSpools,
    listSupplies,
    listSupplyMovements,
    recordSpoolMeasurement,
    recordSupplyMovement,
} from './native';

const movementLabels: Record<SupplyMovementType, string> = {
    purchase: 'Compra (+)',
    consume: 'Consumo (−)',
    adjustment: 'Ajuste (+/−)',
    return: 'Devolução (+)',
    discard: 'Descarte (−)',
};

function errorMessage(error: unknown): string {
    if (error instanceof Error) return error.message;
    return typeof error === 'string' ? error : 'Não foi possível concluir a operação.';
}

export function InventoryWorkspace({canWrite}: {canWrite: boolean}) {
    const [materials, setMaterials] = useState<Material[]>([]);
    const [spools, setSpools] = useState<Spool[]>([]);
    const [selectedSpoolID, setSelectedSpoolID] = useState<string | null>(null);
    const [measurements, setMeasurements] = useState<SpoolMeasurement[]>([]);
    const [gross, setGross] = useState('');
    const [measurementNotes, setMeasurementNotes] = useState('');
    const [supplies, setSupplies] = useState<Supply[]>([]);
    const [selectedSupplyID, setSelectedSupplyID] = useState<string | null>(null);
    const [movements, setMovements] = useState<SupplyMovement[]>([]);
    const [lowInventory, setLowInventory] = useState<LowInventory | null>(null);
    const [supplyName, setSupplyName] = useState('');
    const [supplySKU, setSupplySKU] = useState('');
    const [supplyUnit, setSupplyUnit] = useState('unit');
    const [supplyMinimum, setSupplyMinimum] = useState('0');
    const [supplyReplacementCost, setSupplyReplacementCost] = useState('0');
    const [movementType, setMovementType] = useState<SupplyMovementType>('purchase');
    const [movementQuantity, setMovementQuantity] = useState('');
    const [movementNotes, setMovementNotes] = useState('');
    const [busy, setBusy] = useState(false);
    const [feedback, setFeedback] = useState<string | null>(null);

    const selectedSpool = useMemo(
        () => spools.find((spool) => spool.id === selectedSpoolID) ?? null,
        [spools, selectedSpoolID],
    );
    const selectedSupply = useMemo(
        () => supplies.find((supply) => supply.id === selectedSupplyID) ?? null,
        [supplies, selectedSupplyID],
    );
    const materialNames = useMemo(
        () => new Map(materials.map((material) => [material.id, `${material.manufacturer} ${material.name}`])),
        [materials],
    );

    useEffect(() => {
        let active = true;
        void Promise.all([listMaterials(), listSpools(), listSupplies(), listLowInventory('100')])
            .then(([loadedMaterials, loadedSpools, loadedSupplies, loadedLowInventory]) => {
                if (!active) return;
                setMaterials(loadedMaterials);
                setSpools(loadedSpools);
                setSupplies(loadedSupplies);
                setLowInventory(loadedLowInventory);
                setSelectedSpoolID(loadedSpools[0]?.id ?? null);
                setSelectedSupplyID(loadedSupplies[0]?.id ?? null);
            })
            .catch((error: unknown) => {
                if (active) setFeedback(errorMessage(error));
            });
        return () => {
            active = false;
        };
    }, []);

    useEffect(() => {
        if (!selectedSpoolID) return;
        let active = true;
        void listSpoolMeasurements(selectedSpoolID)
            .then((values) => {
                if (active) setMeasurements(values);
            })
            .catch((error: unknown) => {
                if (active) setFeedback(errorMessage(error));
            });
        return () => {
            active = false;
        };
    }, [selectedSpoolID]);

    useEffect(() => {
        if (!selectedSupplyID) return;
        let active = true;
        void listSupplyMovements(selectedSupplyID)
            .then((values) => {
                if (active) setMovements(values);
            })
            .catch((error: unknown) => {
                if (active) setFeedback(errorMessage(error));
            });
        return () => {
            active = false;
        };
    }, [selectedSupplyID]);

    async function weigh(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        if (!selectedSpool) return;
        setBusy(true);
        setFeedback(null);
        try {
            const value = await recordSpoolMeasurement(selectedSpool.id, {
                measured_at: new Date().toISOString(),
                gross_weight_g: gross,
                source: 'manual',
                notes: measurementNotes,
            });
            setMeasurements((current) => [value, ...current]);
            setSpools((current) => current.map((spool) => spool.id === selectedSpool.id
                ? {...spool, current_remaining_weight_g: value.derived_remaining_weight_g, last_weighed_at: value.measured_at}
                : spool));
            setGross('');
            setMeasurementNotes('');
            void listLowInventory('100').then(setLowInventory).catch(() => {
                setFeedback('Pesagem registrada, mas não foi possível atualizar os avisos de estoque baixo.');
            });
        } catch (error: unknown) {
            setFeedback(errorMessage(error));
        } finally {
            setBusy(false);
        }
    }

    async function addSupply(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        setBusy(true);
        setFeedback(null);
        try {
            const value = await createSupply({
                name: supplyName,
                sku: supplySKU.trim() || null,
                unit: supplyUnit,
                replacement_unit_cost_cents: Number(supplyReplacementCost),
                minimum_quantity: supplyMinimum,
                notes: '',
            });
            setSupplies((current) => [...current, value].sort((left, right) => left.name.localeCompare(right.name)));
            setSelectedSupplyID(value.id);
            setSupplyName('');
            setSupplySKU('');
            void listLowInventory('100').then(setLowInventory).catch(() => {
                setFeedback('Supply cadastrado, mas não foi possível atualizar os avisos de estoque baixo.');
            });
        } catch (error: unknown) {
            setFeedback(errorMessage(error));
        } finally {
            setBusy(false);
        }
    }

    async function moveSupply(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        if (!selectedSupply) return;
        setBusy(true);
        setFeedback(null);
        try {
            const value = await recordSupplyMovement(selectedSupply.id, {
                type: movementType,
                quantity: movementQuantity,
                occurred_at: new Date().toISOString(),
                notes: movementNotes,
            });
            setMovements((current) => [value, ...current]);
            setMovementQuantity('');
            setMovementNotes('');
            try {
                const [updatedSupplies, updatedLowInventory] = await Promise.all([listSupplies(), listLowInventory('100')]);
                setSupplies(updatedSupplies);
                setLowInventory(updatedLowInventory);
            } catch {
                setFeedback('Movimento registrado, mas não foi possível atualizar o saldo exibido. Reabra a tela antes de registrar outro movimento.');
            }
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
                    <h1 id="inventory-title">Materiais, bobinas e supplies</h1>
                    <p>Saldo físico e histórico auditável sem permitir estoque negativo.</p>
                </div>
            </div>
            {feedback && <p className="feedback feedback--error" role="alert">{feedback}</p>}
            {lowInventory && (lowInventory.spools.length > 0 || lowInventory.supplies.length > 0) && (
                <aside className="low-inventory" aria-labelledby="low-inventory-title">
                    <h2 id="low-inventory-title">Estoque baixo</h2>
                    <p>{lowInventory.spools.length} bobina(s) com até {lowInventory.spool_threshold_g} g · {lowInventory.supplies.length} supply(s) no mínimo.</p>
                    <ul className="low-inventory__list">
                        {lowInventory.spools.map((spool) => <li key={`spool-${spool.id}`}><strong>{spool.code}</strong><span>{spool.current_remaining_weight_g} g restantes</span></li>)}
                        {lowInventory.supplies.map((supply) => <li key={`supply-${supply.id}`}><strong>{supply.name}</strong><span>{supply.current_quantity} {supply.unit} · mínimo {supply.minimum_quantity}</span></li>)}
                    </ul>
                </aside>
            )}

            <h2>Bobinas</h2>
            <div className="inventory-layout">
                <nav className="catalog-list" aria-label="Bobinas">
                    {spools.length === 0 && <p>Nenhuma bobina cadastrada.</p>}
                    {spools.map((spool) => (
                        <button className={`catalog-list__item${spool.id === selectedSpoolID ? ' catalog-list__item--selected' : ''}`} key={spool.id} type="button" onClick={() => setSelectedSpoolID(spool.id)}>
                            <strong>{spool.code}</strong>
                            <span>{materialNames.get(spool.material_id) ?? 'Material desconhecido'} · {spool.status}</span>
                            <small>{spool.current_remaining_weight_g ?? '—'} g restantes</small>
                        </button>
                    ))}
                </nav>
                <article className="catalog-detail">
                    {selectedSpool ? <>
                        <div className="catalog-detail__heading"><div><span className="catalog-status">{selectedSpool.status}</span><h2>{selectedSpool.code}</h2></div></div>
                        <dl className="catalog-detail__facts">
                            <div><dt>Material</dt><dd>{materialNames.get(selectedSpool.material_id) ?? '—'}</dd></div>
                            <div><dt>Tara</dt><dd>{selectedSpool.tare_weight_g} g</dd></div>
                            <div><dt>Saldo atual</dt><dd>{selectedSpool.current_remaining_weight_g ?? 'Sem pesagem'}{selectedSpool.current_remaining_weight_g ? ' g' : ''}</dd></div>
                        </dl>
                        {canWrite && <form className="design-inline-form" onSubmit={weigh}>
                            <label>Peso bruto (g)<input required inputMode="decimal" pattern="[0-9]+([.][0-9]{1,3})?" value={gross} disabled={busy} onChange={(event) => setGross(event.target.value)} /></label>
                            <label>Notas<input value={measurementNotes} maxLength={10000} disabled={busy} onChange={(event) => setMeasurementNotes(event.target.value)} /></label>
                            <button className="button button--primary" disabled={busy} type="submit">Registrar pesagem</button>
                        </form>}
                        <h3>Histórico imutável</h3>
                        {measurements.length === 0 ? <p>Nenhuma pesagem registrada.</p> : <ol className="measurement-history">{measurements.map((measurement) => <li key={measurement.id}><strong>{measurement.derived_remaining_weight_g} g restantes</strong><span>Bruto {measurement.gross_weight_g} g · {new Date(measurement.measured_at).toLocaleString('pt-BR')}</span>{measurement.notes && <small>{measurement.notes}</small>}</li>)}</ol>}
                    </> : <p>Selecione uma bobina.</p>}
                </article>
            </div>

            <div className="inventory-section-heading"><div><h2>Supplies</h2><p>Movimentos usam variação assinada e nunca podem deixar o saldo negativo.</p></div></div>
            {canWrite && <form className="supply-create-form" onSubmit={addSupply}>
                <label>Nome<input required maxLength={200} value={supplyName} disabled={busy} onChange={(event) => setSupplyName(event.target.value)} /></label>
                <label>SKU<input maxLength={100} value={supplySKU} disabled={busy} onChange={(event) => setSupplySKU(event.target.value)} /></label>
                <label>Unidade<input required maxLength={50} value={supplyUnit} disabled={busy} onChange={(event) => setSupplyUnit(event.target.value)} /></label>
                <label>Quantidade mínima<input required inputMode="decimal" pattern="[0-9]+([.][0-9]{1,6})?" value={supplyMinimum} disabled={busy} onChange={(event) => setSupplyMinimum(event.target.value)} /></label>
                <label>Custo unitário (centavos)<input required type="number" min="0" step="1" value={supplyReplacementCost} disabled={busy} onChange={(event) => setSupplyReplacementCost(event.target.value)} /></label>
                <button className="button button--primary" disabled={busy} type="submit">Cadastrar supply</button>
            </form>}
            <div className="inventory-layout">
                <nav className="catalog-list" aria-label="Supplies">
                    {supplies.length === 0 && <p>Nenhum supply cadastrado.</p>}
                    {supplies.map((supply) => <button className={`catalog-list__item${supply.id === selectedSupplyID ? ' catalog-list__item--selected' : ''}`} key={supply.id} type="button" onClick={() => setSelectedSupplyID(supply.id)}><strong>{supply.name}</strong><span>{supply.sku ?? 'Sem SKU'} · {supply.unit}</span><small>{supply.current_quantity} disponíveis · mínimo {supply.minimum_quantity}</small></button>)}
                </nav>
                <article className="catalog-detail">
                    {selectedSupply ? <>
                        <div className="catalog-detail__heading"><div><span className="catalog-status">{selectedSupply.unit}</span><h2>{selectedSupply.name}</h2></div></div>
                        <dl className="catalog-detail__facts">
                            <div><dt>Saldo</dt><dd>{selectedSupply.current_quantity}</dd></div>
                            <div><dt>Mínimo</dt><dd>{selectedSupply.minimum_quantity}</dd></div>
                            <div><dt>Reposição</dt><dd>{selectedSupply.replacement_unit_cost_cents} centavos/{selectedSupply.unit}</dd></div>
                        </dl>
                        {canWrite && <form className="design-inline-form" onSubmit={moveSupply}>
                            <label>Tipo<select value={movementType} disabled={busy} onChange={(event) => setMovementType(event.target.value as SupplyMovementType)}>{Object.entries(movementLabels).map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
                            <label>Variação assinada<input required inputMode="decimal" pattern="-?[0-9]+([.][0-9]{1,6})?" placeholder={movementType === 'consume' || movementType === 'discard' ? '-1' : '1'} value={movementQuantity} disabled={busy} onChange={(event) => setMovementQuantity(event.target.value)} /></label>
                            <label>Notas<input maxLength={10000} value={movementNotes} disabled={busy} onChange={(event) => setMovementNotes(event.target.value)} /></label>
                            <button className="button button--primary" disabled={busy} type="submit">Registrar movimento</button>
                        </form>}
                        <h3>Histórico de movimentos</h3>
                        {movements.length === 0 ? <p>Nenhum movimento registrado.</p> : <ol className="measurement-history">{movements.map((movement) => <li key={movement.id}><strong>{movementLabels[movement.type]} {movement.quantity} {selectedSupply.unit}</strong><span>{new Date(movement.occurred_at).toLocaleString('pt-BR')}</span>{movement.notes && <small>{movement.notes}</small>}</li>)}</ol>}
                    </> : <p>Selecione um supply.</p>}
                </article>
            </div>
        </section>
    );
}
