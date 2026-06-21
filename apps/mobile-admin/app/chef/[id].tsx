import { useState } from 'react';
import { Alert, ScrollView, View } from 'react-native';
import { useLocalSearchParams } from 'expo-router';
import { Screen, Button } from '@homechef/mobile-shared/ui';
import {
  useAdminChefFromCache,
  useVerifyChef,
  useRejectChef,
  useSuspendChef,
} from '../../hooks/useAdminChefs';
import { chefStatus } from '../(tabs)/chefs';
import {
  Badge,
  Card,
  Empty,
  Field,
  ScreenHeader,
  SectionTitle,
} from '../../components/kit';
import { PromptModal } from '../../components/PromptModal';
import { formatINR, formatDate, titleCase, errorMessage } from '../../lib/format';

export default function ChefDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const chef = useAdminChefFromCache(id);
  const verify = useVerifyChef();
  const reject = useRejectChef();
  const suspend = useSuspendChef();
  const [rejectOpen, setRejectOpen] = useState(false);

  if (!chef) {
    return (
      <Screen>
        <ScreenHeader title="Chef" back />
        <Empty
          title="Details unavailable"
          body="Open this chef from the Chefs list to load full details."
        />
      </Screen>
    );
  }

  const status = chefStatus(chef);

  const doVerify = () => {
    Alert.alert('Verify kitchen', `Approve ${chef.businessName}?`, [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Verify',
        onPress: () =>
          verify.mutate(chef.id, {
            onError: (e) => Alert.alert('Failed', errorMessage(e)),
            onSuccess: () => Alert.alert('Done', 'Kitchen verified.'),
          }),
      },
    ]);
  };

  const doSuspend = () => {
    Alert.alert('Suspend kitchen', `Suspend ${chef.businessName}? They will stop receiving orders.`, [
      { text: 'Cancel', style: 'cancel' },
      {
        text: 'Suspend',
        style: 'destructive',
        onPress: () =>
          suspend.mutate(chef.id, {
            onError: (e) => Alert.alert('Failed', errorMessage(e)),
            onSuccess: () => Alert.alert('Done', 'Kitchen suspended.'),
          }),
      },
    ]);
  };

  return (
    <Screen>
      <ScreenHeader title="Chef" back right={<Badge label={status.label} tone={status.tone} />} />
      <ScrollView showsVerticalScrollIndicator={false} contentContainerStyle={{ paddingBottom: 40 }}>
        <Card>
          <Field label="Business name" value={chef.businessName || '—'} />
          <Field label="Owner" value={chef.ownerName || '—'} />
          <Field label="Email" value={chef.ownerEmail || '—'} />
          <Field label="Phone" value={chef.ownerPhone || '—'} />
          <Field
            label="Kitchen type"
            value={chef.kitchenType ? titleCase(chef.kitchenType) : 'Home kitchen'}
          />
          <Field
            label="Cuisines"
            value={chef.cuisines?.length ? chef.cuisines.join(', ') : '—'}
          />
          <Field label="Joined" value={formatDate(chef.createdAt)} />
        </Card>

        <SectionTitle>Performance</SectionTitle>
        <Card>
          <Field label="Total orders" value={String(chef.totalOrders)} />
          <Field label="Total revenue" value={formatINR(chef.totalRevenue)} />
          <Field label="Rating" value={`${chef.rating?.toFixed(1) ?? '0.0'} ★`} />
          <Field label="Menu items" value={String(chef.menuItemCount)} />
          <Field
            label="Availability"
            value={
              <Badge
                label={titleCase(chef.onlineStatus)}
                tone={chef.onlineStatus === 'online' ? 'success' : 'neutral'}
              />
            }
          />
          <Field
            label="Accepting orders"
            value={
              <Badge
                label={chef.acceptingOrders ? 'Yes' : 'No'}
                tone={chef.acceptingOrders ? 'success' : 'warning'}
              />
            }
          />
        </Card>

        <SectionTitle>Actions</SectionTitle>
        <View style={{ gap: 10 }}>
          {!chef.isVerified ? (
            <>
              <Button
                label="Verify kitchen"
                onPress={doVerify}
                loading={verify.isPending}
              />
              <Button
                label="Reject application"
                variant="destructive"
                onPress={() => setRejectOpen(true)}
                loading={reject.isPending}
              />
            </>
          ) : chef.isActive ? (
            <Button
              label="Suspend kitchen"
              variant="destructive"
              onPress={doSuspend}
              loading={suspend.isPending}
            />
          ) : (
            <Button label="Verify kitchen" onPress={doVerify} loading={verify.isPending} />
          )}
        </View>
      </ScrollView>

      <PromptModal
        visible={rejectOpen}
        title="Reject application"
        message={`Tell ${chef.businessName} why their application was rejected.`}
        placeholder="Reason for rejection…"
        confirmLabel="Reject"
        destructive
        submitting={reject.isPending}
        onClose={() => setRejectOpen(false)}
        onConfirm={(reason) =>
          reject.mutate(
            { chefId: chef.id, reason },
            {
              onError: (e) => Alert.alert('Failed', errorMessage(e)),
              onSuccess: () => {
                setRejectOpen(false);
                Alert.alert('Done', 'Application rejected.');
              },
            }
          )
        }
      />
    </Screen>
  );
}
