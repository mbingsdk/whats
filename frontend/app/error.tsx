"use client";
export default function ErrorBoundary({ reset }: { reset: () => void }) {
  return <main role="alert"><h1>Unable to load this view</h1>
    <p>Try again. No action is submitted automatically.</p>
    <button onClick={reset}>Try again</button>
  </main>;
}
