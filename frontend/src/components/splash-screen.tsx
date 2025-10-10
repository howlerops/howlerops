import React from 'react'

export function SplashScreen() {
  return (
    <div className="theme-howler flex h-screen w-screen items-center justify-center bg-bg">
      <div className="text-center">
        {/* Logo */}
        <div className="mb-8">
          <svg width="128" height="128" viewBox="0 0 256 256" className="mx-auto">
            <defs>
              <linearGradient id="bronzeGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" style={{stopColor:'#c08b3e',stopOpacity:1}} />
                <stop offset="100%" style={{stopColor:'#d7a75e',stopOpacity:1}} />
              </linearGradient>
              <linearGradient id="steelGradient" x1="0%" y1="0%" x2="100%" y2="100%">
                <stop offset="0%" style={{stopColor:'#6ea2c9',stopOpacity:1}} />
                <stop offset="100%" style={{stopColor:'#9bc0db',stopOpacity:1}} />
              </linearGradient>
            </defs>
            
            {/* Background circle */}
            <circle cx="128" cy="128" r="120" fill="#0b1217" stroke="url(#bronzeGradient)" strokeWidth="4"/>
            
            {/* Wolf head silhouette */}
            <path d="M 80 140 
                     Q 70 120 80 100
                     Q 90 90 110 95
                     Q 130 85 150 95
                     Q 170 90 180 100
                     Q 190 120 180 140
                     Q 175 150 170 155
                     Q 165 160 160 158
                     Q 155 155 150 150
                     Q 140 145 130 150
                     Q 120 145 110 150
                     Q 105 155 100 158
                     Q 95 160 90 155
                     Q 85 150 80 140 Z" 
                  fill="url(#bronzeGradient)"/>
            
            {/* Wolf eye */}
            <polygon points="120,120 125,115 130,120 125,125" fill="#0b1217"/>
            
            {/* Circuit traces */}
            <path d="M 60 60 
                     Q 70 80 90 90
                     Q 100 95 110 90" 
                  stroke="url(#steelGradient)" 
                  strokeWidth="3" 
                  fill="none"/>
            
            <path d="M 200 200 
                     Q 190 180 170 170
                     Q 160 165 150 170" 
                  stroke="url(#steelGradient)" 
                  strokeWidth="3" 
                  fill="none"/>
            
            {/* Circuit nodes */}
            <circle cx="60" cy="60" r="4" fill="url(#steelGradient)"/>
            <circle cx="90" cy="90" r="3" fill="url(#steelGradient)"/>
            <circle cx="200" cy="200" r="4" fill="url(#steelGradient)"/>
            <circle cx="170" cy="170" r="3" fill="url(#steelGradient)"/>
          </svg>
        </div>
        
        {/* App Name */}
        <h1 className="text-gradient-ember text-4xl font-bold mb-2">
          HowlerOps
        </h1>
        
        {/* Subtitle */}
        <p className="text-mute text-lg mb-8">
          HowlerOps
        </p>
        
        {/* Loading indicator */}
        <div className="flex justify-center">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-accent"></div>
        </div>
      </div>
    </div>
  )
}
