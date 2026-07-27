import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { motion } from 'framer-motion';
import { ChefHat, Loader2, MailCheck } from 'lucide-react';
import { Button } from '@/shared/components/ui';
import { fadeInLeft } from '@/shared/utils/animations';
import { sendPasswordReset } from '@/features/auth/services/auth-service';

// Password recovery for customers. Mirrors the mobile app's
// (auth)/forgot-password screen, which is the reference implementation.
//
// The whole flow is one call to sendPasswordReset() — our API mints a
// single-use, 15-minute link and mails it from the verified platform sender.
// See the comment on that function for why Firebase's own mailer is avoided.
export default function ForgotPasswordPage() {
  const [email, setEmail] = useState('');
  const [sending, setSending] = useState(false);
  const [sent, setSent] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (sending) return;
    setSending(true);
    setError(null);
    try {
      await sendPasswordReset(email.trim());
      // Success is claimed for any well-formed address, including one with no
      // account. The API answers identically either way on purpose — telling
      // the user "no such account" would turn this form into a way to test
      // whether an address is registered.
      setSent(true);
    } catch (err: unknown) {
      // sendPasswordReset only rejects on transport/infrastructure failure, and
      // its message is already written for a human. Never surface a raw status
      // code or exception text here.
      setError(
        err instanceof Error
          ? err.message
          : "We couldn't send the reset email just now. Please try again in a few minutes.",
      );
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="flex min-h-screen bg-paper">
      <motion.div
        initial="hidden"
        animate="visible"
        variants={fadeInLeft}
        transition={{ duration: 0.5 }}
        className="flex flex-1 flex-col justify-center px-4 py-12 sm:px-6 lg:flex-none lg:px-20 xl:px-24"
      >
        <div className="mx-auto w-full max-w-sm lg:w-96">
          <Link to="/" className="group inline-flex items-center gap-2">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-herb shadow-1 transition-shadow group-hover:shadow-2">
              <ChefHat aria-hidden="true" className="h-5 w-5 text-paper" />
            </div>
            <span className="font-display text-2xl font-semibold text-ink">Fe3dr</span>
          </Link>

          {sent ? (
            <div className="mt-8">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-herb/10">
                <MailCheck aria-hidden="true" className="h-6 w-6 text-herb" />
              </div>
              <h2 className="mt-4 font-display text-display-xs text-ink">Check your inbox</h2>
              <p className="mt-2 text-ink-soft">
                If an account exists for <span className="font-medium text-ink">{email.trim()}</span>,
                we've sent a link to reset your password. It expires in 15 minutes and can
                only be used once.
              </p>
              <p className="mt-4 text-sm text-ink-muted">
                Nothing arrived? Check your spam folder, or{' '}
                <button
                  type="button"
                  onClick={() => setSent(false)}
                  className="font-medium text-herb transition-colors hover:text-herb"
                >
                  try a different address
                </button>
                .
              </p>
              <Button asChild variant="outline" className="mt-8 w-full rounded-full">
                <Link to="/login">Back to sign in</Link>
              </Button>
            </div>
          ) : (
            <>
              <div className="mt-8">
                <h2 className="font-display text-display-xs text-ink">Reset your password</h2>
                <p className="mt-2 text-ink-soft">
                  Enter the email you signed up with and we'll send you a link to set a
                  new password.
                </p>
              </div>

              {error && (
                <div
                  role="alert"
                  className="mt-6 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive"
                >
                  {error}
                </div>
              )}

              <form onSubmit={handleSubmit} className="mt-6 space-y-4">
                <div>
                  <label
                    htmlFor="reset-email"
                    className="block text-sm font-medium text-ink-soft"
                  >
                    Email
                  </label>
                  <input
                    id="reset-email"
                    type="email"
                    required
                    autoComplete="email"
                    value={email}
                    onChange={e => setEmail(e.target.value)}
                    className="mt-1 block w-full rounded-lg border border-mist-strong px-3 py-2.5 text-ink shadow-1 placeholder:text-ink-muted focus-visible:border-herb focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-herb/30"
                    placeholder="you@example.com"
                  />
                </div>

                <Button
                  type="submit"
                  variant="primary"
                  disabled={sending}
                  className="w-full rounded-full"
                >
                  {sending ? (
                    <>
                      <Loader2 aria-hidden="true" className="mr-2 h-4 w-4 animate-spin" />
                      Sending…
                    </>
                  ) : (
                    'Send reset link'
                  )}
                </Button>
              </form>

              <p className="mt-6 text-sm text-ink-soft">
                Remembered it?{' '}
                <Link
                  to="/login"
                  className="font-medium text-herb transition-colors hover:text-herb"
                >
                  Back to sign in
                </Link>
              </p>
            </>
          )}
        </div>
      </motion.div>
    </div>
  );
}
