import {FormEvent, useEffect, useState} from 'react';
import {LaborRate, LaborSuggestion, listLaborRates, saveLaborRate, suggestLaborRate} from './native';

const activities = [
    ['setup', 'Preparação'], ['material_handling', 'Manuseio de material'],
    ['support_removal', 'Remoção de suportes'], ['finishing', 'Acabamento'],
    ['painting', 'Pintura'], ['assembly', 'Montagem'], ['packaging', 'Embalagem'],
    ['modeling', 'Modelagem'], ['customization', 'Personalização'],
    ['consulting', 'Consultoria'], ['other', 'Outra'],
];

// Parse decimal BRL text directly to cents, without floating point arithmetic.
function cents(value: string): string {
    const match = /^(\d+)(?:[.,](\d{1,2}))?$/.exec(value.trim());
    if (!match) throw new Error('Informe um valor em reais com até duas casas decimais.');
    const amount = BigInt(match[1]) * 100n + BigInt((match[2] ?? '').padEnd(2, '0'));
    if (amount > 9223372036854775807n) throw new Error('Valor monetário acima do limite permitido.');
    return amount.toString();
}

function reais(value: string): string {
    const amount = BigInt(value);
    return `${amount / 100n},${(amount % 100n).toString().padStart(2, '0')}`;
}

function message(error: unknown): string {
    return error instanceof Error ? error.message : typeof error === 'string' ? error : 'Não foi possível concluir a operação.';
}

export function LaborWorkspace({canManage}: {canManage: boolean}) {
    const [rates, setRates] = useState<LaborRate[]>([]);
    const [selectedID, setSelectedID] = useState('');
    const [name, setName] = useState('');
    const [activity, setActivity] = useState('setup');
    const [active, setActive] = useState(true);
    const [hourly, setHourly] = useState('');
    const [compensation, setCompensation] = useState('');
    const [overhead, setOverhead] = useState('');
    const [hours, setHours] = useState('');
    const [utilization, setUtilization] = useState('');
    const [suggestion, setSuggestion] = useState<LaborSuggestion | null>(null);
    const [busy, setBusy] = useState(false);
    const [loading, setLoading] = useState(true);
    const [feedback, setFeedback] = useState('');

    useEffect(() => {
        let current = true;
        void listLaborRates().then((items) => { if (current) setRates(items); })
            .catch((error: unknown) => { if (current) setFeedback(message(error)); })
            .finally(() => { if (current) setLoading(false); });
        return () => { current = false; };
    }, []);

    function selectRate(id: string) {
        const rate = rates.find((item) => item.id === id);
        setSelectedID(id); setName(rate?.name ?? ''); setActivity(rate?.activity_type ?? 'setup');
        setActive(rate?.active ?? true); setHourly(rate ? reais(rate.cost_hourly_rate_cents) : '');
        setSuggestion(null); setFeedback('');
    }

    async function calculate(event: FormEvent<HTMLFormElement>) {
        event.preventDefault(); setBusy(true); setFeedback(''); setSuggestion(null);
        try {
            // Percent text has the same exact two-place scale as BRL input.
            const bps = BigInt(cents(utilization));
            if (bps <= 0n || bps > 10000n) throw new Error('A utilização deve ser maior que 0% e no máximo 100%.');
            const result = await suggestLaborRate({
                target_monthly_compensation_cents: cents(compensation),
                monthly_labor_overhead_cents: cents(overhead),
                available_hours_per_month: hours.trim().replace(',', '.'),
                productive_utilization_bps: Number(bps),
            });
            setSuggestion(result);
        } catch (error: unknown) { setFeedback(message(error)); }
        finally { setBusy(false); }
    }

    async function save(event: FormEvent<HTMLFormElement>) {
        event.preventDefault(); setBusy(true); setFeedback('');
        try {
            const rate = await saveLaborRate(selectedID, {name, activity_type: activity, active, cost_hourly_rate_cents: cents(hourly)});
            setRates((items) => [...items.filter((item) => item.id !== rate.id), rate]);
            setSelectedID(rate.id);
            setFeedback('Taxa interna salva. Os registros anteriores de Jobs mantêm seus custos originais.');
        } catch (error: unknown) { setFeedback(message(error)); }
        finally { setBusy(false); }
    }

    return <section className="labor-workspace" aria-labelledby="labor-title">
        <h2 id="labor-title">Custo da hora humana</h2>
        <p>Estime o custo interno do tempo produtivo da oficina. O preço cobrado por serviços é definido separadamente.</p>
        {loading && <p role="status">Carregando taxas…</p>}
        <div className="labor-workspace__columns">
            <form onSubmit={(event) => void calculate(event)}>
                <fieldset disabled={busy || loading}>
                    <legend>Assistente opcional</legend>
                    <label>Remuneração mensal desejada (R$)<input inputMode="decimal" maxLength={24} value={compensation} required onChange={(e) => {setCompensation(e.target.value); setSuggestion(null);}} /></label>
                    <label>Despesas mensais de mão de obra (R$)<input inputMode="decimal" maxLength={24} value={overhead} required onChange={(e) => {setOverhead(e.target.value); setSuggestion(null);}} /></label>
                    <label>Horas disponíveis por mês<input inputMode="decimal" maxLength={22} value={hours} required onChange={(e) => {setHours(e.target.value); setSuggestion(null);}} /></label>
                    <label>Utilização produtiva (%)<input inputMode="decimal" maxLength={6} value={utilization} required onChange={(e) => {setUtilization(e.target.value); setSuggestion(null);}} /></label>
                    <p>Nem toda hora disponível é produtiva. Informe zero nas despesas quando não houver custo adicional.</p>
                    <button className="button button--secondary" type="submit">Calcular sugestão</button>
                </fieldset>
                {suggestion && <div role="status">
                    <p>Horas produtivas: <strong>{suggestion.productive_hours.replace(/\.?0+$/, '').replace('.', ',')}</strong></p>
                    <p>Custo interno sugerido: <strong>R$ {reais(suggestion.internal_hourly_cost_cents)}/h</strong></p>
                    {canManage && <button className="button button--secondary" type="button" disabled={busy} onClick={() => setHourly(reais(suggestion.internal_hourly_cost_cents))}>Usar sugestão no formulário</button>}
                </div>}
            </form>
            <form onSubmit={(event) => void save(event)}>
                <fieldset disabled={busy || loading}>
                    <legend>Taxas internas</legend>
                    <label>Taxa<select value={selectedID} onChange={(e) => selectRate(e.target.value)}>
                        <option value="">{canManage ? 'Nova taxa' : 'Selecione uma taxa'}</option>
                        {rates.map((rate) => <option key={rate.id} value={rate.id}>{rate.name}{rate.active ? '' : ' (inativa)'}</option>)}
                    </select></label>
                    <label>Nome<input value={name} required maxLength={200} disabled={!canManage} onChange={(e) => setName(e.target.value)} /></label>
                    <label>Atividade<select value={activity} disabled={!canManage} onChange={(e) => setActivity(e.target.value)}>{activities.map(([value, label]) => <option key={value} value={value}>{label}</option>)}</select></label>
                    <label>Custo interno por hora (R$)<input inputMode="decimal" value={hourly} maxLength={24} required disabled={!canManage} onChange={(e) => setHourly(e.target.value)} /></label>
                    <label className="labor-workspace__checkbox"><input type="checkbox" checked={active} disabled={!canManage} onChange={(e) => setActive(e.target.checked)} />Taxa ativa</label>
                    {canManage && <><p>Você pode ajustar o valor ou informar uma taxa manual, sem usar o assistente. O cálculo não salva automaticamente.</p><button className="button button--primary" type="submit">Salvar taxa interna</button></>}
                </fieldset>
            </form>
        </div>
        {feedback && <p className="feedback" role="status">{feedback}</p>}
    </section>;
}
