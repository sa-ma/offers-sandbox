import React from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter, NavLink, Route, Routes } from 'react-router-dom'
import './index.css'
import Rules from './pages/Rules'
import Live from './pages/Live'

function App() {
  return (
    <BrowserRouter>
      <div className="min-h-screen">
        <header className="bg-white shadow">
          <div className="max-w-6xl mx-auto px-4 py-4 flex items-center justify-between">
            <h1 className="text-xl font-semibold">Offers Sandbox</h1>
            <nav className="flex gap-4">
              <NavLink to="/" end className={({isActive})=>`px-3 py-2 rounded-md text-sm font-medium ${isActive?'bg-gray-900 text-white':'text-gray-700 hover:bg-gray-100'}`}>Rules</NavLink>
              <NavLink to="/live" className={({isActive})=>`px-3 py-2 rounded-md text-sm font-medium ${isActive?'bg-gray-900 text-white':'text-gray-700 hover:bg-gray-100'}`}>Live Feed</NavLink>
            </nav>
          </div>
        </header>
        <main className="max-w-6xl mx-auto px-4 py-6">
          <Routes>
            <Route path="/" element={<Rules/>} />
            <Route path="/live" element={<Live/>} />
          </Routes>
        </main>
      </div>
    </BrowserRouter>
  )
}

createRoot(document.getElementById('root')!).render(<App />)
