// react-native-razorpay ships only `declare module 'react-native-razorpay';`
// (implicit any), which trips noImplicitAny. Declare the slice we use.
declare module 'react-native-razorpay' {
  export interface RazorpaySuccessResponse {
    razorpay_payment_id: string;
    razorpay_order_id: string;
    razorpay_signature: string;
  }
  const RazorpayCheckout: {
    open(options: Record<string, unknown>): Promise<RazorpaySuccessResponse>;
  };
  export default RazorpayCheckout;
}
