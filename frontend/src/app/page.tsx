export default function HomePage() {
  return (
    <main className="min-h-screen p-8">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-4xl font-bold mb-4">AIAUDITOR</h1>
        <p className="text-lg text-gray-600">
          Autonomous AI-powered cybersecurity audit reasoning engine
        </p>
        <div className="mt-8 grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="p-4 border rounded-lg">
            <h2 className="font-semibold">NIS 2 Compliance</h2>
            <p className="text-sm text-gray-500">Article 21(a-j) automated audit</p>
          </div>
          <div className="p-4 border rounded-lg">
            <h2 className="font-semibold">Evidence Management</h2>
            <p className="text-sm text-gray-500">Automated collection and scoring</p>
          </div>
          <div className="p-4 border rounded-lg">
            <h2 className="font-semibold">Audit Trail</h2>
            <p className="text-sm text-gray-500">Immutable hash-chained log</p>
          </div>
        </div>
      </div>
    </main>
  );
}

