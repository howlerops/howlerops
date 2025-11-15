import React from 'react';
import iconLight from '@/assets/howlerops-icon-light.png';
import iconDark from '@/assets/howlerops-icon-dark.png';
import iconDefault from '@/assets/howlerops-icon.png';

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
  const getIconSrc = () => {
    switch (variant) {
      case 'light':
        return iconLight;
      case 'dark':
        return iconDark;
      case 'logo':
        return iconDefault;
      default:
        return iconDefault;
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