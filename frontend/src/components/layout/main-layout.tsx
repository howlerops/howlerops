import { Header } from "./header"
import { Sidebar } from "./sidebar"

interface MainLayoutProps {
  children: React.ReactNode
}

export function MainLayout({ children }: MainLayoutProps) {
  return (
    <div className="flex h-screen flex-col">
      <Header />
      <div className="flex flex-1 min-h-0 overflow-hidden">
        <Sidebar />
        <main className="flex-1 bg-bg relative flex min-h-0 overflow-hidden">
          <div className="flex-1 flex min-h-0 flex-col overflow-hidden">
            {children}
          </div>
        </main>
      </div>
    </div>
  )
}
