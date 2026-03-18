import { useParams, useNavigate, Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  ArrowLeft,
  Clock,
  Users,
  IndianRupee,
  Pencil,
  Star,
  ShoppingBag,
  CheckCircle,
  AlertCircle,
  Loader2,
} from 'lucide-react';
import { apiClient } from '@/shared/services/api-client';
import { Button } from '@/shared/components/ui/Button';
import { Badge } from '@/shared/components/ui/Badge';
import { Card } from '@/shared/components/ui/Card';
import type { MenuItem } from '@/shared/types';

export default function MenuItemViewPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const { data: item, isLoading } = useQuery({
    queryKey: ['menu-item', id],
    queryFn: () => apiClient.get<MenuItem>(`/chef/menu/items/${id}`),
    enabled: !!id,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    );
  }

  if (!item) {
    return (
      <div className="py-20 text-center">
        <p className="text-muted-foreground">Menu item not found</p>
        <button onClick={() => navigate('/menu')} className="mt-4 text-sm text-primary hover:underline">
          Back to Menu
        </button>
      </div>
    );
  }

  const menuItem = item as MenuItem;

  return (
    <div className="mx-auto max-w-4xl space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <button onClick={() => navigate('/menu')} className="rounded-lg p-2 hover:bg-secondary transition-colors">
            <ArrowLeft className="h-5 w-5 text-muted-foreground" />
          </button>
          <div>
            <h1 className="text-2xl font-bold text-foreground">{menuItem.name}</h1>
            <div className="flex items-center gap-2 mt-1">
              {menuItem.isApproved ? (
                <Badge variant="success" size="sm"><CheckCircle className="h-3 w-3 mr-1" />Approved</Badge>
              ) : (
                <Badge variant="warning" size="sm"><AlertCircle className="h-3 w-3 mr-1" />Under Review</Badge>
              )}
              {menuItem.isAvailable ? (
                <Badge variant="success" size="sm">Available</Badge>
              ) : (
                <Badge variant="outline" size="sm">Unavailable</Badge>
              )}
              {menuItem.isFeatured && <Badge variant="brand" size="sm">Featured</Badge>}
            </div>
          </div>
        </div>
        <Button variant="outline" size="md" asChild>
          <Link to={`/menu/${id}/edit`}>
            <Pencil className="h-4 w-4 mr-2" />Edit Item
          </Link>
        </Button>
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        {/* Main content */}
        <div className="lg:col-span-2 space-y-6">
          {/* Images */}
          {menuItem.images && menuItem.images.length > 0 ? (
            <Card padding="none">
              <div className="grid grid-cols-2 gap-1">
                {menuItem.images.map((img, idx) => (
                  <div key={img.id} className={`overflow-hidden ${idx === 0 ? 'col-span-2 aspect-video' : 'aspect-square'}`}>
                    <img src={img.url} alt={menuItem.name} className="h-full w-full object-cover" />
                  </div>
                ))}
              </div>
            </Card>
          ) : menuItem.imageUrl ? (
            <Card padding="none">
              <div className="aspect-video overflow-hidden">
                <img src={menuItem.imageUrl} alt={menuItem.name} className="h-full w-full object-cover" />
              </div>
            </Card>
          ) : null}

          {/* Description */}
          <Card>
            <h2 className="text-lg font-semibold text-foreground mb-2">Description</h2>
            <p className="text-muted-foreground">{menuItem.description || 'No description provided'}</p>
          </Card>

          {/* Dietary & Allergens */}
          <Card>
            <h2 className="text-lg font-semibold text-foreground mb-3">Dietary Information</h2>
            <div className="space-y-3">
              {menuItem.dietaryTags && menuItem.dietaryTags.length > 0 && (
                <div>
                  <p className="text-xs text-muted-foreground mb-1.5">Dietary Tags</p>
                  <div className="flex flex-wrap gap-1.5">
                    {menuItem.dietaryTags.map((tag) => (
                      <Badge key={tag} variant="brand" size="sm">{tag}</Badge>
                    ))}
                  </div>
                </div>
              )}
              {menuItem.allergens && menuItem.allergens.length > 0 && (
                <div>
                  <p className="text-xs text-muted-foreground mb-1.5">Allergens</p>
                  <div className="flex flex-wrap gap-1.5">
                    {menuItem.allergens.map((a) => (
                      <Badge key={a} variant="destructive" size="sm">{a}</Badge>
                    ))}
                  </div>
                </div>
              )}
              {(!menuItem.dietaryTags || menuItem.dietaryTags.length === 0) &&
               (!menuItem.allergens || menuItem.allergens.length === 0) && (
                <p className="text-sm text-muted-foreground">No dietary information provided</p>
              )}
            </div>
          </Card>
        </div>

        {/* Sidebar */}
        <div className="space-y-6">
          {/* Pricing */}
          <Card>
            <h2 className="text-lg font-semibold text-foreground mb-3">Pricing</h2>
            <div className="flex items-baseline gap-2">
              <span className="text-3xl font-bold text-primary">₹{menuItem.price.toFixed(2)}</span>
              {(menuItem.comparePrice ?? 0) > 0 && (menuItem.comparePrice ?? 0) > menuItem.price && (
                <span className="text-lg text-muted-foreground line-through">₹{(menuItem.comparePrice ?? 0).toFixed(2)}</span>
              )}
            </div>
          </Card>

          {/* Preparation Details */}
          <Card>
            <h2 className="text-lg font-semibold text-foreground mb-3">Preparation</h2>
            <div className="space-y-3">
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10">
                  <Clock className="h-4 w-4 text-primary" />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Prep Time</p>
                  <p className="text-sm font-medium text-foreground">{menuItem.prepTime} minutes</p>
                </div>
              </div>
              <div className="flex items-center gap-3">
                <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10">
                  <Users className="h-4 w-4 text-primary" />
                </div>
                <div>
                  <p className="text-xs text-muted-foreground">Serves</p>
                  <p className="text-sm font-medium text-foreground">{menuItem.serves} {menuItem.serves === 1 ? 'person' : 'people'}</p>
                </div>
              </div>
              {menuItem.portionSize && (
                <div className="flex items-center gap-3">
                  <div className="flex h-9 w-9 items-center justify-center rounded-lg bg-primary/10">
                    <IndianRupee className="h-4 w-4 text-primary" />
                  </div>
                  <div>
                    <p className="text-xs text-muted-foreground">Portion Size</p>
                    <p className="text-sm font-medium text-foreground">{menuItem.portionSize}</p>
                  </div>
                </div>
              )}
            </div>
          </Card>

          {/* Stats */}
          <Card>
            <h2 className="text-lg font-semibold text-foreground mb-3">Performance</h2>
            <div className="space-y-2">
              <div className="flex items-center justify-between rounded-lg bg-muted/50 px-3 py-2">
                <div className="flex items-center gap-2"><ShoppingBag className="h-4 w-4 text-muted-foreground" /><span className="text-sm text-muted-foreground">Orders</span></div>
                <span className="text-sm font-medium text-foreground">{(menuItem as unknown as Record<string, number>).totalOrders || 0}</span>
              </div>
              <div className="flex items-center justify-between rounded-lg bg-muted/50 px-3 py-2">
                <div className="flex items-center gap-2"><Star className="h-4 w-4 text-muted-foreground" /><span className="text-sm text-muted-foreground">Rating</span></div>
                <span className="text-sm font-medium text-foreground">{((menuItem as unknown as Record<string, number>).rating || 0).toFixed(1)} ({(menuItem as unknown as Record<string, number>).totalReviews || 0})</span>
              </div>
            </div>
          </Card>

          {/* Item ID */}
          <div className="rounded-lg bg-muted/30 px-4 py-3">
            <p className="text-xs text-muted-foreground">Item ID</p>
            <code className="text-xs font-mono text-foreground">{menuItem.id}</code>
          </div>
        </div>
      </div>
    </div>
  );
}
