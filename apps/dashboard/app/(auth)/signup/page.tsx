import dynamic from "next/dynamic";

const SignupForm = dynamic(() => import("./SignupForm"), { ssr: false });

export default function SignupPage() {
  return <SignupForm />;
}
