// A small controlled modal that collects one line/paragraph of text before
// confirming an action (reject reason, hide reason, request-info note, …).
// Cross-platform — Alert.prompt is iOS-only, so we roll our own.
import { useEffect, useState } from 'react';
import {
  ActivityIndicator,
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native';
import { theme } from '@homechef/mobile-shared/theme';

const c = theme.colors;

export function PromptModal({
  visible,
  title,
  message,
  placeholder,
  confirmLabel = 'Confirm',
  destructive = false,
  required = true,
  multiline = true,
  keyboardType = 'default',
  initialValue = '',
  submitting = false,
  onConfirm,
  onClose,
}: {
  visible: boolean;
  title: string;
  message?: string;
  placeholder?: string;
  confirmLabel?: string;
  destructive?: boolean;
  required?: boolean;
  multiline?: boolean;
  keyboardType?: 'default' | 'numeric';
  initialValue?: string;
  submitting?: boolean;
  onConfirm: (text: string) => void;
  onClose: () => void;
}) {
  const [text, setText] = useState(initialValue);

  useEffect(() => {
    if (visible) setText(initialValue);
  }, [visible, initialValue]);

  const canConfirm = !submitting && (!required || text.trim().length > 0);

  return (
    <Modal visible={visible} transparent animationType="fade" onRequestClose={onClose}>
      <KeyboardAvoidingView
        behavior={Platform.OS === 'ios' ? 'padding' : undefined}
        style={styles.backdrop}
      >
        <Pressable style={styles.backdropPress} onPress={onClose} />
        <View style={styles.sheet}>
          <Text style={styles.title}>{title}</Text>
          {message ? <Text style={styles.message}>{message}</Text> : null}
          <TextInput
            value={text}
            onChangeText={setText}
            placeholder={placeholder}
            placeholderTextColor={c.ink.muted}
            style={[styles.input, multiline && { minHeight: 88, textAlignVertical: 'top' }]}
            multiline={multiline}
            keyboardType={keyboardType}
            autoFocus
          />
          <View style={styles.actions}>
            <Pressable onPress={onClose} style={[styles.btn, styles.btnGhost]} disabled={submitting}>
              <Text style={styles.btnGhostText}>Cancel</Text>
            </Pressable>
            <Pressable
              onPress={() => canConfirm && onConfirm(text.trim())}
              disabled={!canConfirm}
              style={[
                styles.btn,
                destructive ? styles.btnDanger : styles.btnPrimary,
                !canConfirm && styles.btnDisabled,
              ]}
            >
              {submitting ? (
                <ActivityIndicator color={c.paper} size="small" />
              ) : (
                <Text style={styles.btnPrimaryText}>{confirmLabel}</Text>
              )}
            </Pressable>
          </View>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: 'flex-end' },
  backdropPress: {
    position: 'absolute',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(14,14,12,0.45)',
  },
  sheet: {
    backgroundColor: c.paper,
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
    padding: 20,
    paddingBottom: 32,
    gap: 12,
  },
  title: { fontFamily: 'Geist', fontSize: 20, color: c.ink.DEFAULT },
  message: { fontFamily: 'Inter', fontSize: 13, color: c.ink.soft, marginTop: -4 },
  input: {
    backgroundColor: c.bone,
    borderRadius: 10,
    padding: 12,
    fontFamily: 'Inter',
    fontSize: 15,
    color: c.ink.DEFAULT,
  },
  actions: { flexDirection: 'row', gap: 10, marginTop: 4 },
  btn: {
    flex: 1,
    height: 48,
    borderRadius: 12,
    alignItems: 'center',
    justifyContent: 'center',
  },
  btnGhost: { backgroundColor: c.bone },
  btnGhostText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: c.ink.DEFAULT },
  btnPrimary: { backgroundColor: c.ink.DEFAULT },
  btnDanger: { backgroundColor: c.destructive.DEFAULT },
  btnPrimaryText: { fontFamily: 'Inter-SemiBold', fontSize: 15, color: c.paper },
  btnDisabled: { opacity: 0.4 },
});
