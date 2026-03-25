import { AuthForm } from "../../../../components/auth/auth-form";
import { type AppLocale } from "../../../../i18n/routing";
import { redirectIfAuthenticated } from "../../../../lib/auth";

interface RegisterPageProps {
  params: Promise<{
    locale: AppLocale;
  }>;
}

export default async function RegisterPage({ params }: RegisterPageProps) {
  const { locale } = await params;
  await redirectIfAuthenticated(locale);

  return (
    <main className="flex min-h-screen items-center justify-center px-6 py-20">
      <AuthForm mode="register" />
    </main>
  );
}
