import {FormEvent, useEffect, useMemo, useState} from 'react';
import {
    Job,
    JobMaterialUsage,
    JobMaterialUsageInput,
    JobMaterialUsageSummary,
    MaterialRole,
    MeasurementSource,
    Spool,
    createJobMaterialUsage,
    deleteJobMaterialUsage,
    listJobMaterialUsage,
    listJobs,
    listSpools,
    updateJobMaterialUsage,
} from './native';

const emptySummary: JobMaterialUsageSummary = {items: [], total_planned_grams: '0', total_actual_grams: '0'};
const roleLabels: Record<MaterialRole, string> = {model: 'Modelo', support: 'Suporte', purge: 'Purga', other: 'Outro'};
const sourceLabels: Record<MeasurementSource, string> = {
    slicer: 'Slicer', spool_weight_delta: 'Diferença de peso da bobina', manual: 'Manual', printer: 'Impressora', estimated: 'Estimado',
};

function errorMessage(error: unknown): string {
    if (error instanceof Error) return error.message;
    return typeof error === 'string' ? error : 'Não foi possível concluir a operação.';
}

export function JobWorkspace({canWrite}: {canWrite: boolean}) {
    const [jobs, setJobs] = useState<Job[]>([]);
    const [spools, setSpools] = useState<Spool[]>([]);
    const [selectedJobID, setSelectedJobID] = useState<string | null>(null);
    const [summary, setSummary] = useState<JobMaterialUsageSummary>(emptySummary);
    const [editingID, setEditingID] = useState<string | null>(null);
    const [spoolID, setSpoolID] = useState('');
    const [role, setRole] = useState<MaterialRole>('model');
    const [plannedGrams, setPlannedGrams] = useState('');
    const [actualGrams, setActualGrams] = useState('');
    const [source, setSource] = useState<MeasurementSource>('slicer');
    const [busy, setBusy] = useState(false);
    const [feedback, setFeedback] = useState<string | null>(null);

    const selectedJob = useMemo(() => jobs.find((job) => job.id === selectedJobID) ?? null, [jobs, selectedJobID]);
    const spoolNames = useMemo(() => new Map(spools.map((spool) => [spool.id, spool.code])), [spools]);
    const editable = canWrite && !!selectedJob && ['draft', 'prepared', 'printing', 'awaiting_review'].includes(selectedJob.status);

    useEffect(() => {
        let active = true;
        void Promise.all([listJobs(), listSpools()]).then(([loadedJobs, loadedSpools]) => {
            if (!active) return;
            setJobs(loadedJobs);
            setSpools(loadedSpools);
            setSelectedJobID(loadedJobs[0]?.id ?? null);
            setSpoolID(loadedSpools[0]?.id ?? '');
        }).catch((error: unknown) => { if (active) setFeedback(errorMessage(error)); });
        return () => { active = false; };
    }, []);

    useEffect(() => {
        if (!selectedJobID) return;
        let active = true;
        void listJobMaterialUsage(selectedJobID)
            .then((value) => { if (active) setSummary(value); })
            .catch((error: unknown) => { if (active) setFeedback(errorMessage(error)); });
        return () => { active = false; };
    }, [selectedJobID]);

    function resetForm() {
        setEditingID(null);
        setSpoolID(spools[0]?.id ?? '');
        setRole('model');
        setPlannedGrams('');
        setActualGrams('');
        setSource('slicer');
    }

    function selectJob(jobID: string) {
        resetForm();
        setSummary(emptySummary);
        setSelectedJobID(jobID);
    }

    function edit(value: JobMaterialUsage) {
        setEditingID(value.id);
        setSpoolID(value.spool_id);
        setRole(value.role);
        setPlannedGrams(value.planned_grams);
        setActualGrams(value.actual_grams ?? '');
        setSource(value.measurement_source);
    }

    async function save(event: FormEvent<HTMLFormElement>) {
        event.preventDefault();
        if (!selectedJobID) return;
        const input: JobMaterialUsageInput = {
            spool_id: spoolID, role, planned_grams: plannedGrams,
            actual_grams: actualGrams.trim() ? actualGrams : null,
            planned_meters: null, actual_meters: null, measurement_source: source,
            historical_material_cost_cents: null, replacement_material_cost_cents: null,
        };
        setBusy(true);
        setFeedback(null);
        try {
            if (editingID) await updateJobMaterialUsage(selectedJobID, editingID, input);
            else await createJobMaterialUsage(selectedJobID, input);
            setSummary(await listJobMaterialUsage(selectedJobID));
            resetForm();
        } catch (error: unknown) {
            setFeedback(errorMessage(error));
        } finally {
            setBusy(false);
        }
    }

    async function remove(usageID: string) {
        if (!selectedJobID) return;
        setBusy(true);
        setFeedback(null);
        try {
            await deleteJobMaterialUsage(selectedJobID, usageID);
            setSummary(await listJobMaterialUsage(selectedJobID));
            if (editingID === usageID) resetForm();
        } catch (error: unknown) {
            setFeedback(errorMessage(error));
        } finally {
            setBusy(false);
        }
    }

    return (
        <section className="job-workspace" aria-labelledby="jobs-title">
            <div className="catalog-workspace__toolbar">
                <div><p className="connection-card__eyebrow">Produção</p><h1 id="jobs-title">Materiais dos Jobs</h1><p>Planeje e registre o consumo real por bobina e função.</p></div>
            </div>
            {feedback && <p className="feedback feedback--error" role="alert">{feedback}</p>}
            <div className="catalog-layout">
                <nav className="catalog-list" aria-label="Jobs">
                    {jobs.length === 0 && <p>Nenhum Job cadastrado.</p>}
                    {jobs.map((job) => <button className={`catalog-list__item${job.id === selectedJobID ? ' catalog-list__item--selected' : ''}`} key={job.id} type="button" onClick={() => selectJob(job.id)}><strong>{job.code}</strong><span>{job.purpose} · {job.status}</span><small>{job.planned_quantity} unidade(s)</small></button>)}
                </nav>
                <article className="catalog-detail">
                    {!selectedJob ? <p>Selecione um Job.</p> : <>
                        <div className="catalog-detail__heading"><div><span className="catalog-status">{selectedJob.status}</span><h2>{selectedJob.code}</h2></div></div>
                        <dl className="catalog-detail__facts job-material-totals">
                            <div><dt>Total planejado</dt><dd>{summary.total_planned_grams} g</dd></div>
                            <div><dt>Total real</dt><dd>{summary.total_actual_grams} g</dd></div>
                        </dl>
                        {editable && <form className="design-inline-form job-material-form" onSubmit={save}>
                            <label>Bobina<select required value={spoolID} disabled={busy} onChange={(event) => setSpoolID(event.target.value)}>{spools.map((spool) => <option value={spool.id} key={spool.id}>{spool.code}</option>)}</select></label>
                            <label>Função<select value={role} disabled={busy} onChange={(event) => setRole(event.target.value as MaterialRole)}>{Object.entries(roleLabels).map(([value, label]) => <option value={value} key={value}>{label}</option>)}</select></label>
                            <label>Planejado (g)<input required inputMode="decimal" pattern="[0-9]+([.][0-9]{1,6})?" value={plannedGrams} disabled={busy} onChange={(event) => setPlannedGrams(event.target.value)} /></label>
                            <label>Real (g)<input inputMode="decimal" pattern="[0-9]+([.][0-9]{1,6})?" value={actualGrams} disabled={busy} onChange={(event) => setActualGrams(event.target.value)} /></label>
                            <label>Fonte<select value={source} disabled={busy} onChange={(event) => setSource(event.target.value as MeasurementSource)}>{Object.entries(sourceLabels).map(([value, label]) => <option value={value} key={value}>{label}</option>)}</select></label>
                            <div className="job-material-form__actions"><button className="button button--primary" disabled={busy || !spoolID} type="submit">{editingID ? 'Atualizar uso' : 'Adicionar uso'}</button>{editingID && <button className="button button--secondary" disabled={busy} type="button" onClick={resetForm}>Cancelar</button>}</div>
                        </form>}
                        {!editable && canWrite && <p className="connection-card__hint">Este Job está encerrado; seus usos de material são somente leitura.</p>}
                        <h3>Usos registrados</h3>
                        {summary.items.length === 0 ? <p>Nenhum material registrado.</p> : <ol className="measurement-history job-material-list">{summary.items.map((usage) => <li key={usage.id}><strong>{spoolNames.get(usage.spool_id) ?? usage.spool_id} · {roleLabels[usage.role]}</strong><span>{usage.planned_grams} g planejados · {usage.actual_grams ?? '—'} g reais</span><small>Fonte: {sourceLabels[usage.measurement_source]}</small>{editable && <div className="job-material-list__actions"><button className="button button--secondary" type="button" disabled={busy} onClick={() => edit(usage)}>Editar</button><button className="button button--secondary" type="button" disabled={busy || !['draft', 'prepared'].includes(selectedJob.status)} onClick={() => void remove(usage.id)}>Excluir</button></div>}</li>)}</ol>}
                    </>}
                </article>
            </div>
        </section>
    );
}
