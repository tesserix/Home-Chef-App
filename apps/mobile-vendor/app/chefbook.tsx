// ChefBook authoring — write culinary articles customers read on the web and
// in the customer app.
//
// The editor composes the API's typed blocks directly rather than wrapping a
// rich text field: there's no HTML round-trip, and what the chef arranges here
// is exactly what both readers render.

import { useState } from 'react';
import { Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { BookOpen, ChevronLeft, Plus, Trash2 } from 'lucide-react-native';
import { theme } from '@homechef/mobile-shared/theme';
import {
  Button,
  EmptyState,
  KeyboardAwareScrollView,
  Skeleton,
  useToast,
} from '@homechef/mobile-shared/ui';
import {
  useMyArticles,
  useSaveArticle,
  useDeleteArticle,
  chefBookErrorMessage,
  type ArticleBlock,
  type BlockType,
} from '../hooks/useChefBook';

const BLOCK_TYPES: { type: BlockType; label: string }[] = [
  { type: 'paragraph', label: 'Paragraph' },
  { type: 'heading', label: 'Heading' },
  { type: 'image', label: 'Image' },
  { type: 'quote', label: 'Quote' },
  { type: 'list', label: 'List' },
];

function emptyBlock(type: BlockType): ArticleBlock {
  return type === 'list' ? { type, items: [''] } : { type, text: '' };
}

export default function ChefBookScreen() {
  const { data, isLoading } = useMyArticles();
  const save = useSaveArticle();
  const remove = useDeleteArticle();
  const { show: showToast } = useToast();

  const [writing, setWriting] = useState(false);
  const [title, setTitle] = useState('');
  const [cover, setCover] = useState('');
  const [tags, setTags] = useState('');
  const [blocks, setBlocks] = useState<ArticleBlock[]>([emptyBlock('paragraph')]);

  const articles = data?.data ?? [];

  const reset = () => {
    setWriting(false);
    setTitle('');
    setCover('');
    setTags('');
    setBlocks([emptyBlock('paragraph')]);
  };

  const setBlock = (i: number, patch: Partial<ArticleBlock>) =>
    setBlocks((prev) => prev.map((b, idx) => (idx === i ? { ...b, ...patch } : b)));

  const submit = (status: 'draft' | 'published') => {
    if (!title.trim()) {
      showToast({ message: 'Give your article a title', tone: 'error' });
      return;
    }
    // Drop blocks left empty rather than sending them — the API rejects the
    // whole article on an empty block, which would be a confusing failure for
    // a paragraph the chef simply didn't fill in.
    const cleaned = blocks
      .map((b) =>
        b.type === 'list'
          ? { ...b, items: (b.items ?? []).map((i) => i.trim()).filter(Boolean) }
          : b,
      )
      .filter((b) =>
        b.type === 'image'
          ? !!b.url?.trim()
          : b.type === 'list'
            ? (b.items ?? []).length > 0
            : !!b.text?.trim(),
      );

    if (cleaned.length === 0) {
      showToast({ message: 'Add some content before saving', tone: 'error' });
      return;
    }

    save.mutate(
      {
        title: title.trim(),
        cover: cover.trim(),
        blocks: cleaned,
        tags: tags.split(',').map((t) => t.trim()).filter(Boolean),
        status,
      },
      {
        onSuccess: () => {
          showToast({
            message: status === 'published' ? 'Article published' : 'Draft saved',
            tone: 'success',
          });
          reset();
        },
        onError: (err) => showToast({ message: chefBookErrorMessage(err), tone: 'error' }),
      },
    );
  };

  return (
    <SafeAreaView style={styles.screen} edges={['top']}>
      <View style={styles.header}>
        <Pressable
          onPress={() => (writing ? reset() : router.back())}
          hitSlop={8}
          accessibilityRole="button"
          accessibilityLabel={writing ? 'Discard draft' : 'Go back'}
        >
          <ChevronLeft size={24} color={theme.colors.ink.DEFAULT} />
        </Pressable>
        <Text style={styles.headerTitle}>ChefBook</Text>
        <View style={styles.headerSpacer} />
      </View>

      {writing ? (
        <KeyboardAwareScrollView contentContainerStyle={styles.body}>
          <Text style={styles.label}>Title</Text>
          <TextInput
            value={title}
            onChangeText={setTitle}
            placeholder="Dalma, three ways"
            placeholderTextColor={theme.colors.ink.muted}
            style={styles.input}
          />

          <Text style={styles.label}>Cover image URL (optional)</Text>
          <TextInput
            value={cover}
            onChangeText={setCover}
            placeholder="Leave blank to use your first image"
            placeholderTextColor={theme.colors.ink.muted}
            autoCapitalize="none"
            style={styles.input}
          />

          {blocks.map((b, i) => (
            <View key={i} style={styles.block}>
              <View style={styles.blockHeader}>
                <View style={styles.typeRow}>
                  {BLOCK_TYPES.map((t) => (
                    <Pressable
                      key={t.type}
                      onPress={() => setBlock(i, emptyBlock(t.type))}
                      accessibilityRole="button"
                      accessibilityState={{ selected: b.type === t.type }}
                      style={[styles.typeChip, b.type === t.type && styles.typeChipActive]}
                    >
                      <Text
                        style={[
                          styles.typeChipText,
                          b.type === t.type && styles.typeChipTextActive,
                        ]}
                      >
                        {t.label}
                      </Text>
                    </Pressable>
                  ))}
                </View>
                <Pressable
                  onPress={() => setBlocks((prev) => prev.filter((_, idx) => idx !== i))}
                  disabled={blocks.length === 1}
                  hitSlop={8}
                  accessibilityRole="button"
                  accessibilityLabel={`Remove block ${i + 1}`}
                  style={blocks.length === 1 && styles.disabled}
                >
                  <Trash2 size={18} color={theme.colors.ink.soft} />
                </Pressable>
              </View>

              {b.type === 'image' ? (
                <>
                  <TextInput
                    value={b.url ?? ''}
                    onChangeText={(v) => setBlock(i, { url: v })}
                    placeholder="Image URL"
                    placeholderTextColor={theme.colors.ink.muted}
                    autoCapitalize="none"
                    style={styles.input}
                  />
                  <TextInput
                    value={b.caption ?? ''}
                    onChangeText={(v) => setBlock(i, { caption: v })}
                    placeholder="Caption (optional)"
                    placeholderTextColor={theme.colors.ink.muted}
                    style={styles.input}
                  />
                </>
              ) : b.type === 'list' ? (
                <>
                  {(b.items ?? []).map((item, j) => (
                    <TextInput
                      key={j}
                      value={item}
                      onChangeText={(v) =>
                        setBlock(i, {
                          items: (b.items ?? []).map((it, idx) => (idx === j ? v : it)),
                        })
                      }
                      placeholder={`Item ${j + 1}`}
                      placeholderTextColor={theme.colors.ink.muted}
                      style={styles.input}
                    />
                  ))}
                  <Button
                    variant="ghost"
                    label="Add item"
                    onPress={() => setBlock(i, { items: [...(b.items ?? []), ''] })}
                  />
                </>
              ) : (
                <TextInput
                  value={b.text ?? ''}
                  onChangeText={(v) => setBlock(i, { text: v })}
                  placeholder={b.type === 'heading' ? 'Section heading' : 'Write here…'}
                  placeholderTextColor={theme.colors.ink.muted}
                  multiline={b.type !== 'heading'}
                  style={[styles.input, b.type !== 'heading' && styles.textarea]}
                />
              )}
            </View>
          ))}

          <Button
            variant="ghost"
            label="Add block"
            onPress={() => setBlocks((prev) => [...prev, emptyBlock('paragraph')])}
          />

          <Text style={styles.label}>Tags (comma separated)</Text>
          <TextInput
            value={tags}
            onChangeText={setTags}
            placeholder="odia, dal, weeknight"
            placeholderTextColor={theme.colors.ink.muted}
            autoCapitalize="none"
            style={styles.input}
          />

          <Text style={styles.note}>
            ChefBook is for food and cooking — articles that aren&apos;t about either are declined.
          </Text>

          <View style={styles.actions}>
            <Button
              variant="secondary"
              label="Save draft"
              onPress={() => submit('draft')}
              disabled={save.isPending}
            />
            <Button label="Publish" onPress={() => submit('published')} loading={save.isPending} />
          </View>
        </KeyboardAwareScrollView>
      ) : (
        <ScrollView contentContainerStyle={styles.body}>
          <Text style={styles.intro}>
            Write about your food — recipes, methods, the story behind a dish. Customers read these
            on the web and in the app.
          </Text>

          <Button label="Write an article" onPress={() => setWriting(true)} />

          {isLoading ? (
            <View style={styles.skeletons}>
              <Skeleton height={72} />
              <Skeleton height={72} />
            </View>
          ) : articles.length === 0 ? (
            <EmptyState
              icon={<BookOpen size={28} color={theme.colors.ink.soft} />}
              title="Nothing written yet"
              body="Your first recipe is a good place to start."
            />
          ) : (
            <View style={styles.list}>
              {articles.map((a) => (
                <View key={a.id} style={styles.row}>
                  <View style={styles.rowBody}>
                    <Text style={styles.rowTitle} numberOfLines={1}>
                      {a.title}
                    </Text>
                    <Text style={styles.rowMeta}>
                      {a.status === 'published' ? 'Published' : 'Draft'} · {a.readingMinutes} min ·{' '}
                      {a.reactionsTotal} reactions · {a.commentsCount} comments
                    </Text>
                  </View>
                  <Pressable
                    onPress={() =>
                      remove.mutate(a.id, {
                        onSuccess: () =>
                          showToast({ message: 'Article deleted', tone: 'success' }),
                      })
                    }
                    hitSlop={8}
                    accessibilityRole="button"
                    accessibilityLabel={`Delete ${a.title}`}
                  >
                    <Trash2 size={18} color={theme.colors.ink.soft} />
                  </Pressable>
                </View>
              ))}
            </View>
          )}
        </ScrollView>
      )}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: theme.colors.paper },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    paddingHorizontal: 16,
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: theme.colors.mist.DEFAULT,
  },
  headerTitle: {
    flex: 1,
    textAlign: 'center',
    fontSize: 17,
    fontWeight: '600',
    color: theme.colors.ink.DEFAULT,
  },
  headerSpacer: { width: 24 },
  body: { padding: 16, gap: 12, paddingBottom: 40 },
  intro: { color: theme.colors.ink.soft, fontSize: 14, lineHeight: 20 },
  label: { color: theme.colors.ink.soft, fontSize: 13, fontWeight: '600', marginTop: 4 },
  input: {
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: 8,
    paddingHorizontal: 12,
    paddingVertical: 10,
    minHeight: 44,
    color: theme.colors.ink.DEFAULT,
    fontSize: 15,
    backgroundColor: theme.colors.paper,
  },
  textarea: { minHeight: 96, textAlignVertical: 'top' },
  block: {
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: 10,
    padding: 12,
    gap: 10,
  },
  blockHeader: { flexDirection: 'row', alignItems: 'center', gap: 8 },
  typeRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 6, flex: 1 },
  typeChip: {
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: 999,
    paddingHorizontal: 10,
    paddingVertical: 6,
  },
  typeChipActive: {
    borderColor: theme.colors.ink.DEFAULT,
    backgroundColor: theme.colors.ink.DEFAULT,
  },
  typeChipText: { color: theme.colors.ink.soft, fontSize: 12 },
  typeChipTextActive: { color: theme.colors.paper },
  disabled: { opacity: 0.4 },
  note: { color: theme.colors.ink.muted, fontSize: 12, marginTop: 4 },
  actions: { flexDirection: 'row', justifyContent: 'flex-end', gap: 10, marginTop: 8 },
  skeletons: { gap: 10, marginTop: 8 },
  list: { gap: 10, marginTop: 8 },
  row: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    borderWidth: 1,
    borderColor: theme.colors.mist.DEFAULT,
    borderRadius: 10,
    padding: 12,
  },
  rowBody: { flex: 1, gap: 3 },
  rowTitle: { color: theme.colors.ink.DEFAULT, fontSize: 15, fontWeight: '600' },
  rowMeta: { color: theme.colors.ink.soft, fontSize: 12 },
});
