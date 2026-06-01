// Extracts a user-presentable error message from any error coming out of an
// axios call. Falls back to the provided default if neither a server error
// payload nor a JS Error message is available.
//
// Use this everywhere a mutation's catch block shows an Alert — never
// surface the raw "Request failed with status code 400" to users.
//
// Example:
//   } catch (error: unknown) {
//     Alert.alert('Save failed', getServerErrorMessage(error, 'Try again.'));
//   }
export function getServerErrorMessage(error: unknown, fallback: string): string {
  const serverMessage = (error as { response?: { data?: { error?: string; message?: string } } } | null)
    ?.response?.data;
  if (serverMessage?.error) return serverMessage.error;
  if (serverMessage?.message) return serverMessage.message;
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}
