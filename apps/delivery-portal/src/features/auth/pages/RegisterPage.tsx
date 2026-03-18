import { Navigate } from 'react-router-dom';

// Registration is not available - redirect to login
export default function RegisterPage() {
  return <Navigate to="/login" replace />;
}
