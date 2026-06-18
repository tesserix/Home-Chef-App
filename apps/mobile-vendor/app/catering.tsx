// Chef catering (#55) — open requests → submit quote → booked events → complete.
// Mirrors the web chef CateringPage tabs and the customer-side flow end to end.

import { useMemo, useState } from 'react';
import {
  ActivityIndicator,
  Alert,
  Modal,
  Pressable,
  ScrollView,
  StyleSheet,
  Switch,
  Text,
  TextInput,
  View,
} from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { ChevronLeft, X } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import { Button } from '@homechef/mobile-shared/ui';
import {
  useAvailableCateringRequests,
  useCateringBookings,
  useCompleteCateringBooking,
  useMyCateringQuotes,
  useSubmitCateringQuote,
  type CateringBooking,
  type CateringQuote,
  type CateringRequest,
} from '../hooks/useCateringVendor';

type TabKey = 'open' | 'quotes' | 'bookings';

function money(n: number): string {
  return `₹${Math.round(n).toLocaleString('en-IN')}`;
}
function fmtDate(d?: string): string {
  if (!d) return '';
  return new Date(d).toLocaleDateString(undefined, { day: 'numeric', month: 'short', year: 'numeric' });
}

// ─── Status chip ────────────────────────────────────────────────────────────

function StatusChip({ label, tone }: { label: string; tone: 'neutral' | 'accent' | 'success' }) {
  const bg =
    tone === 'success' ? theme.colors.success.tint : tone === 'accent' ? theme.colors.herb.tint : theme.colors.bone;
  const fg =
    tone === 'success' ? theme.colors.success.soft : tone === 'accent' ? theme.colors.herb.soft : theme.colors.ink.soft;
  return (
    <View style={[styles.chip, { backgroundColor: bg }]}>
      <Text style={[styles.chipText, { color: fg }]}>{label}</Text>
    </View>
  );
}

function quoteStatusChip(status: CateringQuote['status']) {
  switch (status) {
    case 'accepted':
      return <StatusChip label="Accepted" tone="success" />;
    case 'rejected':
      return <StatusChip label="Declined" tone="neutral" />;
    case 'expired':
      return <StatusChip label="Expired" tone="neutral" />;
    default:
      return <StatusChip label="Awaiting response" tone="accent" />;
  }
}

// ─── Tags row ───────────────────────────────────────────────────────────────

function Tags({ items }: { items?: string[] }) {
  if (!items?.length) return null;
  return (
    <View style={styles.tagRow}>
      {items.map((t) => (
        <View key={t} style={styles.tag}>
          <Text style={styles.tagText}>{t}</Text>
        </View>
      ))}
    </View>
  );
}

// ─── Open request card ──────────────────────────────────────────────────────

function RequestCard({ request, onQuote }: { request: CateringRequest; onQuote: () => void }) {
  return (
    <View style={styles.card}>
      <View style={styles.rowBetween}>
        <Text style={styles.cardTitle}>{request.eventType}</Text>
        <Text style={styles.metaStrong}>{request.guestCount} guests</Text>
      </View>
      <Text style={styles.caption}>
        {fmtDate(request.eventDate)}
        {request.eventTime ? ` · ${request.eventTime}` : ''} · {request.city}, {request.state}
      </Text>
      {request.budget != null && request.budget > 0 ? (
        <Text style={styles.metaLine}>Budget: {money(request.budget)}</Text>
      ) : null}
      <Tags items={request.cuisineTypes} />
      <Tags items={request.dietaryNeeds} />
      {request.menuStyle ? <Text style={styles.metaLine}>Menu style: {request.menuStyle}</Text> : null}
      {request.description ? <Text style={styles.descLine}>{request.description}</Text> : null}
      <View style={{ marginTop: theme.spacing[3] }}>
        <Button label="Submit a quote" variant="primary" onPress={onQuote} />
      </View>
    </View>
  );
}

// ─── My quote card ──────────────────────────────────────────────────────────

function QuoteCard({ quote }: { quote: CateringQuote }) {
  return (
    <View style={styles.card}>
      <View style={styles.rowBetween}>
        <Text style={styles.cardTitle}>{money(quote.totalPrice)}</Text>
        {quoteStatusChip(quote.status)}
      </View>
      <Text style={styles.caption}>
        {money(quote.pricePerPerson)}/person · deposit {money(quote.depositAmount)}
      </Text>
      {quote.menuItems?.length ? <Tags items={quote.menuItems} /> : null}
      {quote.validUntil ? (
        <Text style={styles.metaLine}>Valid until {fmtDate(quote.validUntil)}</Text>
      ) : null}
    </View>
  );
}

// ─── Booking card ───────────────────────────────────────────────────────────

function BookingCard({ booking, onComplete, busy }: { booking: CateringBooking; onComplete: () => void; busy: boolean }) {
  const { request, quote } = booking;
  const done = request.status === 'completed';
  return (
    <View style={styles.card}>
      <View style={styles.rowBetween}>
        <Text style={styles.cardTitle}>{request.eventType}</Text>
        <StatusChip label={done ? 'Completed' : 'Confirmed'} tone="success" />
      </View>
      <Text style={styles.caption}>
        {fmtDate(request.eventDate)}
        {request.eventTime ? ` · ${request.eventTime}` : ''} · {request.guestCount} guests
      </Text>
      <Text style={styles.metaLine}>
        {request.city}, {request.state}
      </Text>
      <Text style={styles.metaLine}>
        {money(quote.totalPrice)} · deposit paid {money(quote.depositAmount)}
      </Text>
      {request.contactName || request.contactPhone ? (
        <Text style={styles.metaLine}>
          Contact: {[request.contactName, request.contactPhone].filter(Boolean).join(' · ')}
        </Text>
      ) : null}
      {!done ? (
        <View style={{ marginTop: theme.spacing[3] }}>
          <Button label="Mark completed" variant="secondary" loading={busy} onPress={onComplete} />
        </View>
      ) : null}
    </View>
  );
}

// ─── Quote form modal ───────────────────────────────────────────────────────

function QuoteModal({
  request,
  onClose,
  onSubmitted,
}: {
  request: CateringRequest;
  onClose: () => void;
  onSubmitted: () => void;
}) {
  const submit = useSubmitCateringQuote();
  const [proposedMenu, setProposedMenu] = useState('');
  const [pricePerPerson, setPricePerPerson] = useState('');
  const [totalPrice, setTotalPrice] = useState('');
  const [dishes, setDishes] = useState('');
  const [deposit, setDeposit] = useState('');
  const [validDays, setValidDays] = useState(7);
  const [includesSetup, setSetup] = useState(false);
  const [includesServing, setServing] = useState(false);
  const [includesCleanup, setCleanup] = useState(false);
  const [includesEquipment, setEquipment] = useState(false);

  // When price-per-person changes, suggest the total (guests × per-person).
  function onPerPerson(v: string) {
    const clean = v.replace(/[^0-9.]/g, '');
    setPricePerPerson(clean);
    const per = parseFloat(clean);
    if (!isNaN(per)) setTotalPrice(String(Math.round(per * request.guestCount)));
  }

  function onSubmit() {
    const per = parseFloat(pricePerPerson);
    const total = parseFloat(totalPrice);
    if (!proposedMenu.trim()) {
      Alert.alert('Add a menu', 'Describe what you would serve.');
      return;
    }
    if (isNaN(per) || per <= 0 || isNaN(total) || total <= 0) {
      Alert.alert('Add pricing', 'Enter a price per person and total.');
      return;
    }
    const dep = parseFloat(deposit);
    submit.mutate(
      {
        requestId: request.id,
        input: {
          proposedMenu: proposedMenu.trim(),
          menuItems: dishes
            .split(',')
            .map((s) => s.trim())
            .filter(Boolean),
          pricePerPerson: per,
          totalPrice: total,
          depositAmount: !isNaN(dep) && dep > 0 ? dep : undefined,
          includesSetup,
          includesServing,
          includesCleanup,
          includesEquipment,
          validDays,
        },
      },
      {
        onSuccess: () => {
          Alert.alert('Quote sent', 'The customer will review your quote.');
          onSubmitted();
        },
        onError: () => Alert.alert('Could not send', 'Please try again.'),
      },
    );
  }

  return (
    <Modal visible animationType="slide" presentationStyle="pageSheet" onRequestClose={onClose}>
      <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
        <View style={styles.header}>
          <Text style={styles.title}>Submit a quote</Text>
          <Pressable onPress={onClose} hitSlop={8} accessibilityLabel="Close">
            <X size={24} color={theme.colors.ink.DEFAULT} />
          </Pressable>
        </View>
        <ScrollView contentContainerStyle={styles.scroll} keyboardShouldPersistTaps="handled">
          <Text style={styles.caption}>
            {request.eventType} · {request.guestCount} guests · {fmtDate(request.eventDate)}
          </Text>

          <Text style={styles.label}>Proposed menu</Text>
          <TextInput
            style={[styles.input, styles.textArea]}
            value={proposedMenu}
            onChangeText={setProposedMenu}
            placeholder="Starters, mains, desserts you'd serve…"
            placeholderTextColor={theme.colors.ink.muted}
            multiline
            textAlignVertical="top"
          />

          <Text style={styles.label}>Dishes (comma-separated)</Text>
          <TextInput
            style={styles.input}
            value={dishes}
            onChangeText={setDishes}
            placeholder="Paneer tikka, Dal makhani, Gulab jamun"
            placeholderTextColor={theme.colors.ink.muted}
          />

          <View style={styles.twoCol}>
            <View style={styles.col}>
              <Text style={styles.label}>Price / person (₹)</Text>
              <TextInput
                style={styles.input}
                value={pricePerPerson}
                onChangeText={onPerPerson}
                placeholder="500"
                placeholderTextColor={theme.colors.ink.muted}
                keyboardType="number-pad"
              />
            </View>
            <View style={styles.col}>
              <Text style={styles.label}>Total (₹)</Text>
              <TextInput
                style={styles.input}
                value={totalPrice}
                onChangeText={(v) => setTotalPrice(v.replace(/[^0-9.]/g, ''))}
                placeholder="25000"
                placeholderTextColor={theme.colors.ink.muted}
                keyboardType="number-pad"
              />
            </View>
          </View>

          <Text style={styles.label}>Deposit to confirm (₹) — defaults to 25%</Text>
          <TextInput
            style={styles.input}
            value={deposit}
            onChangeText={(v) => setDeposit(v.replace(/[^0-9.]/g, ''))}
            placeholder="Leave blank for 25% of total"
            placeholderTextColor={theme.colors.ink.muted}
            keyboardType="number-pad"
          />

          <Text style={styles.label}>Includes</Text>
          {[
            ['Setup', includesSetup, setSetup],
            ['Serving staff', includesServing, setServing],
            ['Cleanup', includesCleanup, setCleanup],
            ['Equipment', includesEquipment, setEquipment],
          ].map(([lbl, val, set]) => (
            <View key={lbl as string} style={styles.toggleRow}>
              <Text style={styles.toggleLabel}>{lbl as string}</Text>
              <Switch
                value={val as boolean}
                onValueChange={set as (v: boolean) => void}
                trackColor={{ true: theme.colors.herb.DEFAULT }}
              />
            </View>
          ))}

          <Text style={styles.label}>Quote valid for</Text>
          <View style={styles.validRow}>
            {[3, 7, 14].map((d) => {
              const sel = validDays === d;
              return (
                <Pressable key={d} onPress={() => setValidDays(d)} accessibilityRole="button" accessibilityState={{ selected: sel }}>
                  <View style={[styles.validChip, sel && styles.validChipSel]}>
                    <Text style={[styles.validChipText, sel && styles.validChipTextSel]}>{d} days</Text>
                  </View>
                </Pressable>
              );
            })}
          </View>

          <View style={{ marginTop: theme.spacing[5] }}>
            <Button label="Send quote" variant="primary" loading={submit.isPending} onPress={onSubmit} />
          </View>
        </ScrollView>
      </SafeAreaView>
    </Modal>
  );
}

// ─── Screen ─────────────────────────────────────────────────────────────────

export default function VendorCateringScreen() {
  const [tab, setTab] = useState<TabKey>('open');
  const [quoteTarget, setQuoteTarget] = useState<CateringRequest | null>(null);

  const requests = useAvailableCateringRequests();
  const quotes = useMyCateringQuotes();
  const bookings = useCateringBookings();
  const complete = useCompleteCateringBooking();

  const openList = useMemo(() => requests.data?.data ?? [], [requests.data]);
  const quoteList = useMemo(() => quotes.data?.data ?? [], [quotes.data]);
  const bookingList = useMemo(() => bookings.data?.data ?? [], [bookings.data]);

  const tabs: { key: TabKey; label: string; count: number }[] = [
    { key: 'open', label: 'Open', count: openList.length },
    { key: 'quotes', label: 'My quotes', count: quoteList.length },
    { key: 'bookings', label: 'Bookings', count: bookingList.length },
  ];

  const loading =
    (tab === 'open' && requests.isLoading) ||
    (tab === 'quotes' && quotes.isLoading) ||
    (tab === 'bookings' && bookings.isLoading);

  function completeBooking(requestId: string) {
    Alert.alert('Mark this event completed?', 'Do this after the event has been catered.', [
      { text: 'Not yet', style: 'cancel' },
      {
        text: 'Complete',
        onPress: () =>
          complete.mutate(requestId, { onError: () => Alert.alert('Could not update', 'Please try again.') }),
      },
    ]);
  }

  return (
    <SafeAreaView style={styles.root} edges={['top', 'left', 'right']}>
      <View style={styles.header}>
        <Pressable onPress={() => router.back()} hitSlop={8} accessibilityLabel="Go back">
          <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} />
        </Pressable>
        <Text style={styles.title}>Catering</Text>
        <View style={{ width: 24 }} />
      </View>

      {/* Segmented tabs */}
      <View style={styles.tabBar}>
        {tabs.map((t) => {
          const sel = tab === t.key;
          return (
            <Pressable key={t.key} style={styles.tabItem} onPress={() => setTab(t.key)} accessibilityRole="tab" accessibilityState={{ selected: sel }}>
              <View style={[styles.tabInner, sel && styles.tabInnerSel]}>
                <Text style={[styles.tabText, sel && styles.tabTextSel]}>
                  {t.label}
                  {t.count > 0 ? ` (${t.count})` : ''}
                </Text>
              </View>
            </Pressable>
          );
        })}
      </View>

      {loading ? (
        <View style={styles.centered}>
          <ActivityIndicator color={theme.colors.herb.DEFAULT} />
        </View>
      ) : (
        <ScrollView contentContainerStyle={styles.scroll}>
          {tab === 'open' &&
            (openList.length === 0 ? (
              <EmptyState text="No open catering requests right now. Check back soon." />
            ) : (
              openList.map((r) => (
                <RequestCard key={r.id} request={r} onQuote={() => setQuoteTarget(r)} />
              ))
            ))}

          {tab === 'quotes' &&
            (quoteList.length === 0 ? (
              <EmptyState text="You haven't sent any quotes yet." />
            ) : (
              quoteList.map((q) => <QuoteCard key={q.id} quote={q} />)
            ))}

          {tab === 'bookings' &&
            (bookingList.length === 0 ? (
              <EmptyState text="No confirmed bookings yet. Accepted quotes appear here once the customer pays the deposit." />
            ) : (
              bookingList.map((b) => (
                <BookingCard
                  key={b.quote.id}
                  booking={b}
                  busy={complete.isPending}
                  onComplete={() => completeBooking(b.request.id)}
                />
              ))
            ))}
        </ScrollView>
      )}

      {quoteTarget ? (
        <QuoteModal
          request={quoteTarget}
          onClose={() => setQuoteTarget(null)}
          onSubmitted={() => {
            setQuoteTarget(null);
            setTab('quotes');
          }}
        />
      ) : null}
    </SafeAreaView>
  );
}

function EmptyState({ text }: { text: string }) {
  return (
    <View style={styles.empty}>
      <Text style={styles.emptyText}>{text}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: theme.colors.bone },
  centered: { flex: 1, alignItems: 'center', justifyContent: 'center' },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[3],
  },
  title: { fontFamily: 'Geist-Bold', fontSize: 20, color: theme.colors.ink.DEFAULT },
  scroll: { padding: theme.spacing[4], gap: theme.spacing[3], paddingBottom: theme.spacing[10] },

  tabBar: {
    flexDirection: 'row',
    marginHorizontal: theme.spacing[4],
    marginBottom: theme.spacing[2],
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.md,
    padding: 4,
    gap: 4,
  },
  tabItem: { flex: 1 },
  tabInner: { paddingVertical: theme.spacing[2], alignItems: 'center', borderRadius: theme.radius.DEFAULT },
  tabInnerSel: { backgroundColor: theme.colors.paper, ...theme.shadow[1] },
  tabText: { fontFamily: 'Inter', fontSize: 14, color: theme.colors.ink.soft },
  tabTextSel: { fontFamily: 'Inter-SemiBold', color: theme.colors.ink.DEFAULT },

  card: {
    backgroundColor: theme.colors.paper,
    borderRadius: theme.radius.md,
    padding: theme.spacing[4],
    ...theme.shadow[1],
  },
  rowBetween: { flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: theme.spacing[2] },
  cardTitle: { fontFamily: 'Inter-SemiBold', fontSize: 16, color: theme.colors.ink.DEFAULT, flexShrink: 1 },
  caption: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.muted, marginTop: 4 },
  metaLine: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.soft, marginTop: 4 },
  metaStrong: { fontFamily: 'Inter-SemiBold', fontSize: 13, color: theme.colors.ink.DEFAULT },
  descLine: { fontFamily: 'Inter', fontSize: 13, color: theme.colors.ink.soft, marginTop: 6, lineHeight: 18 },

  chip: { borderRadius: theme.radius.full, paddingHorizontal: 10, paddingVertical: 3 },
  chipText: { fontFamily: 'Inter-SemiBold', fontSize: 12 },

  tagRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 6, marginTop: 8 },
  tag: { backgroundColor: theme.colors.bone, borderRadius: theme.radius.full, paddingHorizontal: 10, paddingVertical: 4 },
  tagText: { fontFamily: 'Inter', fontSize: 12, color: theme.colors.ink.soft },

  empty: { paddingVertical: theme.spacing[10], paddingHorizontal: theme.spacing[6], alignItems: 'center' },
  emptyText: { fontFamily: 'Inter', fontSize: 14, color: theme.colors.ink.muted, textAlign: 'center', lineHeight: 20 },

  // Quote modal form
  label: {
    fontFamily: 'Inter-SemiBold',
    fontSize: 13,
    color: theme.colors.ink.DEFAULT,
    marginTop: theme.spacing[4],
    marginBottom: theme.spacing[2],
  },
  input: {
    backgroundColor: theme.colors.bone,
    borderRadius: theme.radius.DEFAULT,
    paddingHorizontal: theme.spacing[3],
    paddingVertical: theme.spacing[3],
    fontFamily: 'Inter',
    fontSize: 15,
    color: theme.colors.ink.DEFAULT,
  },
  textArea: { minHeight: 88 },
  twoCol: { flexDirection: 'row', gap: theme.spacing[3] },
  col: { flex: 1 },
  toggleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingVertical: theme.spacing[2],
  },
  toggleLabel: { fontFamily: 'Inter', fontSize: 15, color: theme.colors.ink.DEFAULT },
  validRow: { flexDirection: 'row', gap: theme.spacing[2] },
  validChip: {
    paddingHorizontal: theme.spacing[4],
    paddingVertical: theme.spacing[2],
    borderRadius: theme.radius.full,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
  },
  validChipSel: { borderColor: theme.colors.herb.DEFAULT, backgroundColor: theme.colors.herb.tint },
  validChipText: { fontFamily: 'Inter', fontSize: 14, color: theme.colors.ink.soft },
  validChipTextSel: { fontFamily: 'Inter-SemiBold', color: theme.colors.herb.soft },
});
