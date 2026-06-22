import { ScrollView, Text, View } from 'react-native';
import Constants from 'expo-constants';
import { Screen } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { useAdminSettings } from '../../hooks/useAdminSettings';
import { useAuthStore } from '../../store/auth-store';
import {
  Card,
  ErrorState,
  Field,
  LoadingState,
  ScreenHeader,
  SectionTitle,
} from '../../components/kit';
import { titleCase, errorMessage } from '../../lib/format';

const c = theme.colors;

export default function SettingsScreen() {
  const q = useAdminSettings();
  const user = useAuthStore((s) => s.user);

  // Render flat scalar settings generically; nested config has dedicated
  // surfaces in the web admin that aren't mirrored here yet.
  const entries = q.data
    ? Object.entries(q.data).filter(
        ([, v]) => typeof v === 'string' || typeof v === 'number' || typeof v === 'boolean'
      )
    : [];

  return (
    <Screen>
      <ScreenHeader title="Settings" back subtitle="Platform configuration" />
      <ScrollView showsVerticalScrollIndicator={false} contentContainerStyle={{ paddingBottom: 40 }}>
        <SectionTitle>Platform</SectionTitle>
        {q.isLoading ? (
          <LoadingState label="Loading settings…" />
        ) : q.isError ? (
          <ErrorState message={errorMessage(q.error)} onRetry={() => q.refetch()} />
        ) : entries.length === 0 ? (
          <Card>
            <Text style={{ fontFamily: 'Inter', fontSize: 14, color: c.ink.muted }}>
              No editable platform settings exposed here. Manage detailed configuration (pricing,
              tax, payment gateways, policies) from the web admin portal.
            </Text>
          </Card>
        ) : (
          <Card>
            {entries.map(([k, v]) => (
              <Field key={k} label={titleCase(k)} value={String(v)} />
            ))}
          </Card>
        )}

        <SectionTitle>Account</SectionTitle>
        <Card>
          <Field label="Signed in as" value={user?.email ?? '—'} />
          <Field label="Role" value={titleCase(user?.role ?? 'admin')} />
          <Field label="App version" value={Constants.expoConfig?.version ?? '1.0.0'} />
        </Card>

        <View style={{ height: 8 }} />
        <Text style={{ fontFamily: 'Inter', fontSize: 12, color: c.ink.muted, textAlign: 'center' }}>
          Fe3dr Admin · Home Chef platform
        </Text>
      </ScrollView>
    </Screen>
  );
}
