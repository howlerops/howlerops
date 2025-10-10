import React from 'react';

interface HowlerOpsIconProps {
  size?: number;
  className?: string;
  variant?: 'logo' | 'icon' | 'light' | 'dark';
}

export const HowlerOpsIcon: React.FC<HowlerOpsIconProps> = ({ 
  size = 32, 
  className = '',
  variant = 'icon'
}) => {
  // Use PNG icons for better quality and consistency
  const getIconSrc = () => {
    switch (variant) {
      case 'light':
        return '/src/assets/howlerops-icon-light.png';
      case 'dark':
        return '/src/assets/howlerops-icon-dark.png';
      case 'logo':
        return '/src/assets/howlerops-icon.png';
      default:
        return '/src/assets/howlerops-icon.png';
    }
  };

  return (
    <img 
      src={getIconSrc()}
      width={size} 
      height={size} 
      alt="HowlerOps"
      className={className}
    />
  );
};

export default HowlerOpsIcon;