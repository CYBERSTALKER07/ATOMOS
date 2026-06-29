import { Text, View } from 'react-native';

export type StatusExplain = {
  code: string;
  title: string;
  summary: string;
  next_steps?: string[];
  deep_link?: string;
  recoverable: boolean;
};

export function parseExplainFromPayload(payload: unknown): StatusExplain | null {
  if (!payload || typeof payload !== 'object') return null;
  const explain = (payload as { explain?: StatusExplain }).explain;
  if (!explain || typeof explain.title !== 'string') return null;
  return explain;
}

export function explainFromApiError(body: unknown): StatusExplain | null {
  return parseExplainFromPayload(body);
}

export class ApiExplainError extends Error {
  explain: StatusExplain | null;

  constructor(message: string, explain: StatusExplain | null) {
    super(message);
    this.name = 'ApiExplainError';
    this.explain = explain;
  }
}

export async function readApiError(response: Response): Promise<ApiExplainError> {
  let body: unknown = null;
  try {
    body = await response.clone().json();
  } catch {
    body = null;
  }
  const explain = explainFromApiError(body);
  const record = body && typeof body === 'object' ? (body as Record<string, unknown>) : {};
  const message =
    (typeof record.message === 'string' && record.message) ||
    (typeof record.detail === 'string' && record.detail) ||
    (typeof record.error === 'string' && record.error) ||
    explain?.summary ||
    `HTTP ${response.status}`;
  return new ApiExplainError(message, explain);
}

type ExplainStatusBannerProps = {
  explain?: StatusExplain | null;
  fallbackTitle?: string | null;
  fallbackDetail?: string | null;
};

export function ExplainStatusBanner({
  explain,
  fallbackTitle,
  fallbackDetail,
}: ExplainStatusBannerProps) {
  const title = explain?.title ?? fallbackTitle;
  const summary = explain?.summary ?? fallbackDetail;
  if (!title) return null;
  return (
    <View
      style={{
        borderRadius: 12,
        borderWidth: 1,
        borderColor: 'rgba(239,68,68,0.45)',
        backgroundColor: 'rgba(69,10,10,0.9)',
        padding: 14,
        marginBottom: 12,
      }}
    >
      <Text style={{ color: '#FEE2E2', fontWeight: '700', fontSize: 14 }}>{title}</Text>
      {summary ? (
        <Text style={{ color: '#FECACA', fontSize: 13, marginTop: 6 }}>{summary}</Text>
      ) : null}
      {explain?.next_steps?.map((step) => (
        <Text key={step} style={{ color: '#FECACA', fontSize: 12, marginTop: 4 }}>
          • {step}
        </Text>
      ))}
    </View>
  );
}
