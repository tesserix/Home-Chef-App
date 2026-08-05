import { Navigate } from 'react-router';

// Registration is not available - redirect to login
export default function RegisterPage() {
  return <Navigate to="/login" replace />;
}
