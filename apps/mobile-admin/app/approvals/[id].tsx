import { useState } from 'react';
import { ScrollView, Text, View } from 'react-native';
import { useLocalSearchParams } from 'expo-router';
import { AlertTriangle } from 'lucide-react-native';
import { Screen, Button, useAlert } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import {
  useApprovalDetail,
  useApproveRequest,
  useRejectRequest,
  useRequestMoreInfo,
} from '../../hooks/useAdminApprovals';
import {
  Badge,
  Card,
  ErrorState,
  Field,
  LoadingState,
  ScreenHeader,
  SectionTitle,
} from '../../components/kit';
import { PromptModal } from '../../components/PromptModal';
import { formatDateTime, titleCase, errorMessage } from '../../lib/format';

const c = theme.colors;
type Mode = 'reject' | 'info' | null;

function Warning({ text }: { text: string }) {
  return (
    <View
      style={{
        flexDirection: 'row',
        gap: 8,
        backgroundColor: c.amber.tint,
        borderRadius: 10,
        padding: 12,
        marginBottom: 8,
        alignItems: 'center',
      }}
    >
      <AlertTriangle size={18} color="#7a5a13" />
      <Text style={{ flex: 1, fontFamily: 'Inter-Medium', fontSize: 13, color: '#7a5a13' }}>{text}</Text>
    </View>
  );
}

export default function ApprovalDetailScreen() {
  const { showAlert } = useAlert();
  const { id } = useLocalSearchParams<{ id: string }>();
  const q = useApprovalDetail(id);
  const approve = useApproveRequest();
  const reject = useRejectRequest();
  const reqInfo = useRequestMoreInfo();
  const [mode, setMode] = useState<Mode>(null);

  const a = q.data;
  const isPending = a?.status === 'pending' || a?.status === 'info_requested';
  // Kitchen onboarding: the API refuses to approve while required documents
  // (id_proof, address_proof, fssai_license) are missing — mirror that here.
  const docsMissing =
    (a as { requiredDocsMissing?: string[] } | undefined)?.requiredDocsMissing ?? [];
  const docsIncomplete = a?.type === 'kitchen_onboarding' && docsMissing.length > 0;
  const DOC_LABELS: Record<string, string> = {
    id_proof: 'ID proof',
    address_proof: 'Address proof (gas/electricity bill)',
    fssai_license: 'FSSAI licence',
  };
  const busy = approve.isPending || reject.isPending || reqInfo.isPending;

  const doApprove = () => {
    if (!a) return;
    showAlert('Approve request', a.title || titleCase(a.type), [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Approve',
        onPress: () =>
          approve.mutate(
            { id: a.id },
            {
              onError: (e) => showAlert('Failed', errorMessage(e)),
              onSuccess: () => showAlert('Approved', 'Request approved.'),
            }
          ),
      },
    ]);
  };

  const submitted = a?.submittedData ?? {};
  const submittedEntries = Object.entries(submitted).filter(
    ([, v]) => typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean'
  );

  return (
    <Screen>
      <ScreenHeader title="Request" back right={a ? <Badge label={titleCase(a.status)} tone={a.status === 'approved' ? 'success' : a.status === 'rejected' ? 'danger' : 'warning'} /> : undefined} />
      {q.isLoading ? (
        <LoadingState label="Loading request…" />
      ) : q.isError || !a ? (
        <ErrorState message={errorMessage(q.error)} onRetry={() => q.refetch()} />
      ) : (
        <ScrollView showsVerticalScrollIndicator={false} contentContainerStyle={{ paddingBottom: 40 }}>
          {a.kitchenTypeNonHome ? (
            <Warning text="Submitted kitchen type is NOT a home kitchen — Fe3dr onboards home cooks only." />
          ) : null}
          {a.fssaiLooksCommercial ? (
            <Warning text="FSSAI licence looks like a commercial (State/Central) registration — verify this is a home kitchen." />
          ) : null}
          {docsIncomplete ? (
            <Warning
              text={`Required documents not uploaded yet: ${docsMissing
                .map((d) => DOC_LABELS[d] ?? titleCase(d))
                .join(', ')}. Cannot approve until they are in (30-day window from submission).`}
            />
          ) : null}

          <Card>
            <Text style={{ fontFamily: 'Geist', fontSize: 20, color: c.ink.DEFAULT, marginBottom: 8 }}>
              {a.title || titleCase(a.type)}
            </Text>
            {a.description ? (
              <Text style={{ fontFamily: 'Inter', fontSize: 14, color: c.ink.soft, marginBottom: 8 }}>
                {a.description}
              </Text>
            ) : null}
            <Field label="Type" value={titleCase(a.type)} />
            <Field label="Priority" value={titleCase(a.priority)} />
            <Field label="Submitted" value={formatDateTime(a.createdAt)} />
            {a.reviewedAt ? <Field label="Reviewed" value={formatDateTime(a.reviewedAt)} /> : null}
            {a.adminNotes ? <Field label="Admin notes" value={a.adminNotes} /> : null}
          </Card>

          {submittedEntries.length > 0 ? (
            <>
              <SectionTitle>Submitted details</SectionTitle>
              <Card>
                {submittedEntries.map(([k, v]) => (
                  <Field key={k} label={titleCase(k)} value={String(v)} />
                ))}
              </Card>
            </>
          ) : null}

          {a.documents && a.documents.length > 0 ? (
            <>
              <SectionTitle>Documents ({a.documents.length})</SectionTitle>
              <Card>
                {a.documents.map((d) => (
                  <Field
                    key={d.id}
                    label={titleCase(d.type ?? 'Document')}
                    value={d.fileName ?? d.id}
                  />
                ))}
              </Card>
            </>
          ) : null}

          {isPending ? (
            <>
              <SectionTitle>Decision</SectionTitle>
              <View style={{ gap: 10 }}>
                <Button label="Approve" onPress={doApprove} loading={approve.isPending} disabled={busy || docsIncomplete} />
                <Button
                  label="Request more info"
                  variant="secondary"
                  onPress={() => setMode('info')}
                  disabled={busy}
                />
                <Button
                  label="Reject"
                  variant="destructive"
                  onPress={() => setMode('reject')}
                  disabled={busy}
                />
              </View>
            </>
          ) : null}
        </ScrollView>
      )}

      <PromptModal
        visible={mode === 'reject'}
        title="Reject request"
        message="Add a note explaining the rejection."
        placeholder="Reason…"
        confirmLabel="Reject"
        destructive
        submitting={reject.isPending}
        onClose={() => setMode(null)}
        onConfirm={(notes) =>
          a &&
          reject.mutate(
            { id: a.id, notes },
            {
              onError: (e) => showAlert('Failed', errorMessage(e)),
              onSuccess: () => {
                setMode(null);
                showAlert('Rejected', 'Request rejected.');
              },
            }
          )
        }
      />
      <PromptModal
        visible={mode === 'info'}
        title="Request more info"
        message="Tell the applicant what's missing."
        placeholder="What do you need?"
        confirmLabel="Send"
        submitting={reqInfo.isPending}
        onClose={() => setMode(null)}
        onConfirm={(notes) =>
          a &&
          reqInfo.mutate(
            { id: a.id, notes },
            {
              onError: (e) => showAlert('Failed', errorMessage(e)),
              onSuccess: () => {
                setMode(null);
                showAlert('Sent', 'Information requested.');
              },
            }
          )
        }
      />
    </Screen>
  );
}
