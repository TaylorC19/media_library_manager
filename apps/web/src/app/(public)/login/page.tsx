import { AuthForm } from "../../../components/auth/auth-form";
import { redirectIfAuthenticated } from "../../../lib/auth";

export default async function LoginPage() {
  await redirectIfAuthenticated();

  return (
    <main className="flex min-h-screen items-center justify-center px-6 py-20">
      <AuthForm mode="login" />
    </main>
  );
}
