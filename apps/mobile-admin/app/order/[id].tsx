import { ScrollView, Text, View } from 'react-native';
import { useLocalSearchParams } from 'expo-router';
import { Screen } from '@homechef/mobile-shared/ui';
import { theme } from '@homechef/mobile-shared/theme';
import { useAdminOrder } from '../../hooks/useAdminOrders';
import { orderStatusTone } from '../(tabs)/orders';
import {
  Badge,
  Card,
  ErrorState,
  Field,
  LoadingState,
  ScreenHeader,
  SectionTitle,
} from '../../components/kit';
import { formatINR, formatDateTime, titleCase, errorMessage } from '../../lib/format';

const c = theme.colors;

export default function OrderDetailScreen() {
  const { id } = useLocalSearchParams<{ id: string }>();
  const q = useAdminOrder(id);

  return (
    <Screen>
      <ScreenHeader title="Order" back />
      {q.isLoading ? (
        <LoadingState label="Loading order…" />
      ) : q.isError || !q.data ? (
        <ErrorState message={errorMessage(q.error)} onRetry={() => q.refetch()} />
      ) : (
        <ScrollView showsVerticalScrollIndicator={false} contentContainerStyle={{ paddingBottom: 40 }}>
          <Card>
            <View style={{ flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', marginBottom: 8 }}>
              <Text style={{ fontFamily: 'Geist', fontSize: 20, color: c.ink.DEFAULT }}>
                #{q.data.order.orderNumber || q.data.order.id?.slice(0, 8)}
              </Text>
              <Badge
                label={titleCase(q.data.order.status ?? '')}
                tone={orderStatusTone(q.data.order.status ?? '')}
              />
            </View>
            <Field label="Placed" value={formatDateTime(q.data.order.createdAt)} />
            <Field
              label="Payment"
              value={
                <Badge
                  label={titleCase(q.data.order.paymentStatus ?? 'unknown')}
                  tone={q.data.order.paymentStatus === 'paid' ? 'success' : 'warning'}
                />
              }
            />
            {q.data.order.deliveryAddress ? (
              <Field label="Delivery address" value={q.data.order.deliveryAddress} />
            ) : null}
          </Card>

          {Array.isArray(q.data.order.items) && q.data.order.items.length > 0 ? (
            <>
              <SectionTitle>Items</SectionTitle>
              <Card style={{ paddingVertical: 4 }}>
                {q.data.order.items.map((it, idx) => (
                  <View
                    key={idx}
                    style={{
                      flexDirection: 'row',
                      justifyContent: 'space-between',
                      paddingVertical: 10,
                      borderBottomWidth: idx < q.data!.order.items!.length - 1 ? 1 : 0,
                      borderBottomColor: c.mist.DEFAULT,
                    }}
                  >
                    <Text style={{ fontFamily: 'Inter-Medium', fontSize: 14, color: c.ink.DEFAULT, flex: 1 }}>
                      {it.quantity}× {it.name}
                    </Text>
                    <Text style={{ fontFamily: 'Inter-SemiBold', fontSize: 14, color: c.ink.DEFAULT }}>
                      {formatINR(it.price * it.quantity)}
                    </Text>
                  </View>
                ))}
              </Card>
            </>
          ) : null}

          <SectionTitle>Totals</SectionTitle>
          <Card>
            {q.data.order.subtotal != null ? (
              <Field label="Subtotal" value={formatINR(q.data.order.subtotal)} />
            ) : null}
            {q.data.order.deliveryFee != null ? (
              <Field label="Delivery fee" value={formatINR(q.data.order.deliveryFee)} />
            ) : null}
            {q.data.order.tax != null ? (
              <Field label="Tax" value={formatINR(q.data.order.tax)} />
            ) : null}
            <Field label="Total" value={formatINR(q.data.order.total)} />
          </Card>

          <SectionTitle>Customer</SectionTitle>
          <Card>
            <Field label="Name" value={q.data.customer.name || '—'} />
            <Field label="Email" value={q.data.customer.email || '—'} />
            <Field label="Phone" value={q.data.customer.phone || '—'} />
          </Card>

          <SectionTitle>Chef</SectionTitle>
          <Card>
            <Field label="Kitchen" value={q.data.chef.businessName || '—'} />
            {q.data.chef.city ? <Field label="City" value={q.data.chef.city} /> : null}
          </Card>
        </ScrollView>
      )}
    </Screen>
  );
}
