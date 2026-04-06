// Deep link handler for homechef-customer://payment/result
//
// This screen is the callback_url target from Razorpay hosted checkout.
// Razorpay may append razorpay_payment_id / razorpay_order_id / razorpay_signature
// as query params, but we do NOT rely on them for payment confirmation.
//
// Strategy (Open Question 1 resolution from RESEARCH.md):
//   - Primary confirmation: server-side webhook → order status updated on server
//   - Client detection: checkout.tsx polls GET /v1/orders/:id every 3s for up to 60s
//   - This screen just navigates back so the checkout poller can detect the result

import { useEffect } from 'react';
import { View, Text, ActivityIndicator } from 'react-native';
import { useRouter } from 'expo-router';

export default function PaymentResult() {
  const router = useRouter();

  useEffect(() => {
    // Navigate back to checkout which is already polling for payment status
    router.back();
  }, []);

  return (
    <View className="flex-1 items-center justify-center bg-white">
      <ActivityIndicator size="large" color="#F97316" />
      <Text className="text-gray-500 mt-3 text-sm">Confirming payment...</Text>
    </View>
  );
}
