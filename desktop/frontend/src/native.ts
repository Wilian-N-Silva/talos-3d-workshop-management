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
    role?: string;
    permissions?: string[];
}

export interface WorkshopBranding {
    workshop_name: string;
    logo_data_url?: string;
}

export interface ShellContext {
    authentication: AuthenticationState;
    workshop_name?: string;
    logo_data_url?: string;
    default_theme?: ThemeMode;
}

export type ThemeMode = 'light' | 'dark' | 'system';
export type CatalogPurpose = 'product' | 'prototype' | 'tooling' | 'test' | 'internal' | 'personal';
export type CatalogStatus = 'active' | 'archived';

export interface CatalogItem {
    id: string;
    name: string;
    sku?: string | null;
    description: string;
    purpose: CatalogPurpose;
    sellable: boolean;
    tags: string[];
    status: CatalogStatus;
    created_at: string;
    updated_at: string;
}

export interface CatalogItemInput {
    name: string;
    sku?: string | null;
    description: string;
    purpose: CatalogPurpose;
    sellable: boolean;
    tags: string[];
    status: CatalogStatus;
}

export interface CatalogPage {
    items: CatalogItem[];
    pagination: {limit: number; offset: number; total: number};
}

export interface CatalogPart {
    id: string;
    catalog_item_id: string;
    name: string;
    quantity: number;
    notes: string;
    created_at: string;
    updated_at: string;
}

export interface CatalogPartInput {
    name: string;
    quantity: number;
    notes: string;
}

export type DesignOrigin = 'original' | 'customer' | 'remix' | 'third_party' | 'unknown';
export type DesignFileRole = 'source' | 'mesh' | 'print' | 'preview' | 'documentation' | 'other';

export interface DesignFile {
    file_id: string;
    role: DesignFileRole;
    original_name: string;
    content_type: string;
    size_bytes: number;
    sha256: string;
    created_at: string;
}

export interface DesignVersion {
    id: string;
    catalog_part_id: string;
    version: string;
    notes: string;
    origin: DesignOrigin;
    source_url?: string | null;
    original_author: string;
    license_name: string;
    commercial_use_allowed?: boolean | null;
    attribution_required: boolean;
    attribution_text: string;
    created_by: string;
    created_at: string;
    files: DesignFile[];
}

export interface DesignVersionInput {
    version: string;
    notes: string;
    origin: DesignOrigin;
    source_url?: string | null;
    original_author: string;
    license_name: string;
    commercial_use_allowed?: boolean | null;
    attribution_required: boolean;
    attribution_text: string;
}

export interface Material {
    id: string; manufacturer: string; name: string; material_type: string; color_name: string;
    color_hex?: string | null; nominal_density: string; default_replacement_cost_per_kg_cents: number;
    notes: string; created_at: string; updated_at: string;
}

export type SpoolStatus = 'sealed' | 'open' | 'stored' | 'drying' | 'empty' | 'retired';
export interface Spool {
    id: string; code: string; material_id: string; nominal_net_weight_g: string; tare_weight_g: string;
    gross_weight_at_open_g?: string | null; current_remaining_weight_g?: string | null;
    purchase_cost_cents: number; replacement_cost_per_kg_cents: number; opened_at?: string | null;
    last_weighed_at?: string | null; last_dried_at?: string | null; storage_location: string;
    storage_status: string; lot_number: string; status: SpoolStatus; created_at: string; updated_at: string;
}

export interface SpoolMeasurement {
    id: string; spool_id: string; measured_at: string; gross_weight_g: string;
    derived_remaining_weight_g: string; source: 'manual' | 'imported' | 'other'; notes: string;
    recorded_by: string; created_at: string;
}

export interface MeasurementInput {measured_at: string; gross_weight_g: string; source: 'manual' | 'imported' | 'other'; notes: string;}

export interface Supply {
    id: string;
    name: string;
    sku?: string | null;
    unit: string;
    current_quantity: string;
    replacement_unit_cost_cents: number;
    minimum_quantity: string;
    notes: string;
    created_at: string;
    updated_at: string;
}

export interface SupplyInput {
    name: string;
    sku?: string | null;
    unit: string;
    replacement_unit_cost_cents: number;
    minimum_quantity: string;
    notes: string;
}

export type SupplyMovementType = 'purchase' | 'consume' | 'adjustment' | 'return' | 'discard';

export interface SupplyMovement {
    id: string;
    supply_id: string;
    type: SupplyMovementType;
    quantity: string;
    unit_cost_cents?: number | null;
    reference_type?: string | null;
    reference_id?: string | null;
    occurred_at: string;
    recorded_by: string;
    notes: string;
    created_at: string;
}

export interface SupplyMovementInput {
    type: SupplyMovementType;
    quantity: string;
    unit_cost_cents?: number | null;
    reference_type?: string | null;
    reference_id?: string | null;
    occurred_at: string;
    notes: string;
}

export interface LowInventory {
    spool_threshold_g: string;
    spools: Spool[];
    supplies: Supply[];
}

export type JobStatus = 'draft' | 'prepared' | 'printing' | 'awaiting_review' | 'completed' | 'failed' | 'cancelled';
export interface Job {
    id: string; code: string; catalog_item_id: string; design_version_id: string; printer_id: string;
    order_item_id?: string | null; purpose: string; status: JobStatus; planned_quantity: number;
    good_quantity: number; scrap_quantity: number; quality_status: string; created_at: string; updated_at: string;
}
export type MaterialRole = 'model' | 'support' | 'purge' | 'other';
export type MeasurementSource = 'slicer' | 'spool_weight_delta' | 'manual' | 'printer' | 'estimated';
export interface JobMaterialUsage {
    id: string; print_job_id: string; material_id: string; spool_id: string; role: MaterialRole;
    planned_grams: string; actual_grams?: string | null; planned_meters?: string | null; actual_meters?: string | null;
    measurement_source: MeasurementSource; historical_material_cost_cents?: number | null;
    replacement_material_cost_cents?: number | null; created_at: string; updated_at: string;
}
export interface JobMaterialUsageInput {
    spool_id: string; role: MaterialRole; planned_grams: string; actual_grams?: string | null;
    planned_meters?: string | null; actual_meters?: string | null; measurement_source: MeasurementSource;
    historical_material_cost_cents?: number | null; replacement_material_cost_cents?: number | null;
}
export interface JobMaterialUsageSummary {
    items: JobMaterialUsage[]; total_planned_grams: string; total_actual_grams: string;
}

export interface CatalogBOMItem {
    id: string;
    catalog_item_id: string;
    supply_id: string;
    quantity_per_unit: string;
    waste_percent: string;
    notes: string;
    created_at: string;
    updated_at: string;
}

export interface CatalogBOMItemInput {
    supply_id: string;
    quantity_per_unit: string;
    waste_percent: string;
    notes: string;
}

export interface CatalogBOMPreviewLine extends CatalogBOMItem {
    supply_name: string;
    supply_unit: string;
    replacement_unit_cost_cents: number;
    effective_quantity_per_unit: string;
    exact_replacement_cost_cents_per_unit: string;
}

export interface CatalogBOMPreview {
    items: CatalogBOMPreviewLine[];
    exact_total_replacement_cost_cents: string;
    rounding_applied: boolean;
}

interface NativeApp {
    GetServerConnection(): Promise<ServerConnection>;
    SaveServerConnection(baseURL: string): Promise<ServerConnection>;
    TestServerConnection(baseURL: string): Promise<ConnectionTestResult>;
    Login(emailOrUsername: string, password: string): Promise<AuthenticationState>;
    GetAuthenticationState(): Promise<AuthenticationState>;
    Logout(): Promise<void>;
    GetWorkshopBranding(): Promise<WorkshopBranding>;
    LoadShell(): Promise<ShellContext>;
    ListCatalogItems(): Promise<CatalogPage>;
    CreateCatalogItem(input: CatalogItemInput): Promise<CatalogItem>;
    UpdateCatalogItem(id: string, input: CatalogItemInput): Promise<CatalogItem>;
    ListCatalogParts(itemID: string): Promise<CatalogPart[]>;
    CreateCatalogPart(itemID: string, input: CatalogPartInput): Promise<CatalogPart>;
    ListDesignVersions(partID: string): Promise<DesignVersion[]>;
    CreateDesignVersion(partID: string, input: DesignVersionInput): Promise<DesignVersion>;
    AttachDesignFile(versionID: string, fileID: string, role: DesignFileRole): Promise<DesignFile>;
    GetCatalogBOM(itemID: string): Promise<CatalogBOMPreview>;
    CreateCatalogBOMItem(itemID: string, input: CatalogBOMItemInput): Promise<CatalogBOMItem>;
    UpdateCatalogBOMItem(itemID: string, bomItemID: string, input: CatalogBOMItemInput): Promise<CatalogBOMItem>;
    DeleteCatalogBOMItem(itemID: string, bomItemID: string): Promise<void>;
    ListMaterials(): Promise<Material[]>;
    ListSpools(): Promise<Spool[]>;
    ListSpoolMeasurements(spoolID: string): Promise<SpoolMeasurement[]>;
    RecordSpoolMeasurement(spoolID: string, input: MeasurementInput): Promise<SpoolMeasurement>;
    ListSupplies(): Promise<Supply[]>;
    CreateSupply(input: SupplyInput): Promise<Supply>;
    ListSupplyMovements(supplyID: string): Promise<SupplyMovement[]>;
    RecordSupplyMovement(supplyID: string, input: SupplyMovementInput): Promise<SupplyMovement>;
    ListLowInventory(spoolThresholdG: string): Promise<LowInventory>;
    ListLaborRates(): Promise<LaborRate[]>;
    SaveLaborRate(id: string, input: LaborRateInput): Promise<LaborRate>;
    SuggestLaborRate(input: LaborAssumptions): Promise<LaborSuggestion>;
    ListJobs(): Promise<Job[]>;
    ListJobMaterialUsage(jobID: string): Promise<JobMaterialUsageSummary>;
    CreateJobMaterialUsage(jobID: string, input: JobMaterialUsageInput): Promise<JobMaterialUsage>;
    UpdateJobMaterialUsage(jobID: string, usageID: string, input: JobMaterialUsageInput): Promise<JobMaterialUsage>;
    DeleteJobMaterialUsage(jobID: string, usageID: string): Promise<void>;
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

export async function getWorkshopBranding(): Promise<WorkshopBranding> {
    return app().GetWorkshopBranding();
}

export async function loadShell(): Promise<ShellContext> {
    return app().LoadShell();
}

export async function listCatalogItems(): Promise<CatalogPage> {
    return app().ListCatalogItems();
}

export async function createCatalogItem(input: CatalogItemInput): Promise<CatalogItem> {
    return app().CreateCatalogItem(input);
}

export async function updateCatalogItem(id: string, input: CatalogItemInput): Promise<CatalogItem> {
    return app().UpdateCatalogItem(id, input);
}

export async function listCatalogParts(itemID: string): Promise<CatalogPart[]> {
    return app().ListCatalogParts(itemID);
}

export async function createCatalogPart(itemID: string, input: CatalogPartInput): Promise<CatalogPart> {
    return app().CreateCatalogPart(itemID, input);
}

export async function listDesignVersions(partID: string): Promise<DesignVersion[]> {
    return app().ListDesignVersions(partID);
}

export async function createDesignVersion(partID: string, input: DesignVersionInput): Promise<DesignVersion> {
    return app().CreateDesignVersion(partID, input);
}

export async function attachDesignFile(versionID: string, fileID: string, role: DesignFileRole): Promise<DesignFile> {
    return app().AttachDesignFile(versionID, fileID, role);
}

export async function getCatalogBOM(itemID: string): Promise<CatalogBOMPreview> { return app().GetCatalogBOM(itemID); }
export async function createCatalogBOMItem(itemID: string, input: CatalogBOMItemInput): Promise<CatalogBOMItem> { return app().CreateCatalogBOMItem(itemID, input); }
export async function updateCatalogBOMItem(itemID: string, bomItemID: string, input: CatalogBOMItemInput): Promise<CatalogBOMItem> { return app().UpdateCatalogBOMItem(itemID, bomItemID, input); }
export async function deleteCatalogBOMItem(itemID: string, bomItemID: string): Promise<void> { return app().DeleteCatalogBOMItem(itemID, bomItemID); }

export async function listMaterials(): Promise<Material[]> { return app().ListMaterials(); }
export async function listSpools(): Promise<Spool[]> { return app().ListSpools(); }
export async function listSpoolMeasurements(spoolID: string): Promise<SpoolMeasurement[]> { return app().ListSpoolMeasurements(spoolID); }
export async function recordSpoolMeasurement(spoolID: string, input: MeasurementInput): Promise<SpoolMeasurement> { return app().RecordSpoolMeasurement(spoolID, input); }
export async function listSupplies(): Promise<Supply[]> { return app().ListSupplies(); }
export async function createSupply(input: SupplyInput): Promise<Supply> { return app().CreateSupply(input); }
export async function listSupplyMovements(supplyID: string): Promise<SupplyMovement[]> { return app().ListSupplyMovements(supplyID); }
export async function recordSupplyMovement(supplyID: string, input: SupplyMovementInput): Promise<SupplyMovement> { return app().RecordSupplyMovement(supplyID, input); }
export async function listLowInventory(spoolThresholdG = '100'): Promise<LowInventory> { return app().ListLowInventory(spoolThresholdG); }
export async function listJobs(): Promise<Job[]> { return app().ListJobs(); }
export async function listJobMaterialUsage(jobID: string): Promise<JobMaterialUsageSummary> { return app().ListJobMaterialUsage(jobID); }
export async function createJobMaterialUsage(jobID: string, input: JobMaterialUsageInput): Promise<JobMaterialUsage> { return app().CreateJobMaterialUsage(jobID, input); }
export async function updateJobMaterialUsage(jobID: string, usageID: string, input: JobMaterialUsageInput): Promise<JobMaterialUsage> { return app().UpdateJobMaterialUsage(jobID, usageID, input); }
export async function deleteJobMaterialUsage(jobID: string, usageID: string): Promise<void> { return app().DeleteJobMaterialUsage(jobID, usageID); }

export type LaborRateInput = {name: string; activity_type: string; cost_hourly_rate_cents: string; active: boolean};
export type LaborRate = LaborRateInput & {id: string};
export type LaborAssumptions = {target_monthly_compensation_cents: string; monthly_labor_overhead_cents: string; available_hours_per_month: string; productive_utilization_bps: number};
export type LaborSuggestion = {productive_hours: string; internal_hourly_cost_cents: string};
export async function listLaborRates(): Promise<LaborRate[]> { return app().ListLaborRates(); }
export async function saveLaborRate(id: string, input: LaborRateInput): Promise<LaborRate> { return app().SaveLaborRate(id, input); }
export async function suggestLaborRate(input: LaborAssumptions): Promise<LaborSuggestion> { return app().SuggestLaborRate(input); }
