import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../lib/api';

export interface MenuItemImage {
  id: string;
  url: string;
}

export interface MenuItem {
  id: string;
  name: string;
  description: string;
  price: number;
  category: string;
  isAvailable: boolean;
  isVeg: boolean;
  images: MenuItemImage[];
  preparationTime: number;
}

export interface Category {
  id: string;
  name: string;
}

export interface MenuResponse {
  items: MenuItem[];
  categories: Category[];
}

export interface CreateMenuItemPayload {
  name: string;
  description: string;
  price: number;
  category: string;
  isVeg: boolean;
  preparationTime: number;
}

export interface UpdateMenuItemPayload extends Partial<CreateMenuItemPayload> {
  isAvailable?: boolean;
}

const MENU_KEY = ['chef', 'menu'] as const;

export function useVendorMenu() {
  return useQuery<MenuResponse>({
    queryKey: MENU_KEY,
    queryFn: () => api.get<MenuResponse>('/chef/menu').then((r) => r.data),
    staleTime: 30_000,
  });
}

export function useCreateMenuItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (payload: CreateMenuItemPayload) =>
      api.post<{ item: MenuItem }>('/chef/menu/items', payload).then((r) => r.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MENU_KEY });
    },
  });
}

export function useUpdateMenuItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ itemId, payload }: { itemId: string; payload: UpdateMenuItemPayload }) =>
      api.put<{ item: MenuItem }>(`/chef/menu/items/${itemId}`, payload).then((r) => r.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MENU_KEY });
    },
  });
}

export function useDeleteMenuItem() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (itemId: string) => api.delete(`/chef/menu/items/${itemId}`),
    onMutate: async (itemId) => {
      await queryClient.cancelQueries({ queryKey: MENU_KEY });
      const previous = queryClient.getQueryData<MenuResponse>(MENU_KEY);
      queryClient.setQueryData<MenuResponse>(MENU_KEY, (old) => {
        if (!old) return old;
        return { ...old, items: old.items.filter((item) => item.id !== itemId) };
      });
      return { previous };
    },
    onError: (_err, _itemId, context) => {
      if (context?.previous) {
        queryClient.setQueryData(MENU_KEY, context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: MENU_KEY });
    },
  });
}

export function useUploadMenuPhoto(itemId: string) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (uri: string) => {
      const formData = new FormData();
      const filename = uri.split('/').pop() ?? 'menu-photo.jpg';
      const type = filename.endsWith('.png') ? 'image/png' : 'image/jpeg';
      formData.append('file', { uri, name: filename, type } as unknown as Blob);
      return api.post(`/chef/menu/items/${itemId}/images`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
      });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: MENU_KEY });
    },
  });
}

export function useToggleAvailability() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ itemId, isAvailable }: { itemId: string; isAvailable: boolean }) =>
      api.put(`/chef/menu/items/${itemId}`, { isAvailable }),
    onMutate: async ({ itemId, isAvailable }) => {
      await queryClient.cancelQueries({ queryKey: MENU_KEY });
      const previous = queryClient.getQueryData<MenuResponse>(MENU_KEY);
      queryClient.setQueryData<MenuResponse>(MENU_KEY, (old) => {
        if (!old) return old;
        return {
          ...old,
          items: old.items.map((item) =>
            item.id === itemId ? { ...item, isAvailable } : item,
          ),
        };
      });
      return { previous };
    },
    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(MENU_KEY, context.previous);
      }
    },
    onSettled: () => {
      queryClient.invalidateQueries({ queryKey: MENU_KEY });
    },
  });
}
