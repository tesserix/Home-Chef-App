import { useAuth } from '@/app/providers/AuthProvider';
import { LogOut, Bell, Shield, HelpCircle } from 'lucide-react';

export default function SettingsPage() {
  const { logout } = useAuth();

  return (
    <div className="space-y-6">
      <div className="page-header">
        <h1 className="page-title">Settings</h1>
      </div>

      <div className="space-y-2">
        <button className="flex w-full items-center gap-3 rounded-xl border border-border bg-card p-4 text-left transition-colors hover:bg-secondary">
          <Bell className="h-5 w-5 text-muted-foreground" />
          <div>
            <p className="text-sm font-medium text-foreground">Notifications</p>
            <p className="text-xs text-muted-foreground">Manage push & email notifications</p>
          </div>
        </button>

        <button className="flex w-full items-center gap-3 rounded-xl border border-border bg-card p-4 text-left transition-colors hover:bg-secondary">
          <Shield className="h-5 w-5 text-muted-foreground" />
          <div>
            <p className="text-sm font-medium text-foreground">Privacy & Security</p>
            <p className="text-xs text-muted-foreground">Manage your account security</p>
          </div>
        </button>

        <button className="flex w-full items-center gap-3 rounded-xl border border-border bg-card p-4 text-left transition-colors hover:bg-secondary">
          <HelpCircle className="h-5 w-5 text-muted-foreground" />
          <div>
            <p className="text-sm font-medium text-foreground">Help & Support</p>
            <p className="text-xs text-muted-foreground">Get help with your deliveries</p>
          </div>
        </button>

        <button
          onClick={logout}
          className="flex w-full items-center gap-3 rounded-xl border border-destructive/30 bg-card p-4 text-left transition-colors hover:bg-destructive/5"
        >
          <LogOut className="h-5 w-5 text-destructive" />
          <div>
            <p className="text-sm font-medium text-destructive">Sign Out</p>
            <p className="text-xs text-muted-foreground">Sign out of your account</p>
          </div>
        </button>
      </div>
    </div>
  );
}
