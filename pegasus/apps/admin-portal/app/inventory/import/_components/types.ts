export type ImportSessionStatus =
  | 'INITIALIZED'
  | 'UPLOADED'
  | 'DISCOVERING'
  | 'DISCOVERED'
  | 'MAPPING_REQUIRED'
  | 'APPROVED'
  | 'APPLYING'
  | 'APPLIED'
  | 'FAILED';

export type ImportWizardStep = 'selection' | 'mapping' | 'review' | 'finalize';

export interface SupplierImportSession {
  supplier_id: string;
  session_id: string;
  status: ImportSessionStatus;
  file_name: string;
  total_rows: number;
  error_summary?: Record<string, unknown> | null;
  created_at: string;
  updated_at?: string;
  gcs_path?: string;
}

export interface UploadTicketResponse {
  session_id: string;
  supplier_id: string;
  status: ImportSessionStatus;
  file_name: string;
  upload_url: string;
  gcs_path: string;
  content_type: string;
  expires_in_seconds: number;
  max_file_size_bytes: number;
}

export interface MappingCandidate {
  source_column: string;
  target_field: string;
  confidence: number;
  reason?: string;
  deterministic?: boolean;
}

export interface MappingAnomaly {
  kind: string;
  column?: string;
  detail: string;
  severity?: string;
}

export interface MappingDocument {
  mappings: MappingCandidate[];
  anomalies?: MappingAnomaly[];
  model?: string;
  usage?: {
    prompt_tokens?: number;
    completion_tokens?: number;
    total_tokens?: number;
  };
  generated_at?: string;
}

export interface MappingResponse {
  supplier_id: string;
  session_id: string;
  mapping_json?: MappingDocument | null;
  created_at?: string;
  updated_at?: string;
}

export interface SupplierImportStagedRow {
  supplier_id: string;
  session_id: string;
  row_index: number;
  raw_data?: Record<string, unknown> | null;
  cleaned_data?: Record<string, unknown> | null;
  validation_errors?: string[];
  is_new_product?: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface RowsResponse {
  session_id: string;
  limit: number;
  offset: number;
  has_more: boolean;
  next_offset: number;
  rows_returned: number;
  data: SupplierImportStagedRow[];
}

export interface ResolvedMappingLink {
  sourceColumn: string;
  targetField: string | null;
  confidence: number;
  reason?: string;
  deterministic?: boolean;
  manual?: boolean;
  ignored?: boolean;
}
