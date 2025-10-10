import React from 'react';
import { HowlerOpsIcon } from '@/components/ui/HowlerOpsIcon';

// Example usage of HowlerOpsIcon component
export const IconExamples = () => {
  return (
    <div className="p-8 space-y-8">
      <h2 className="text-2xl font-bold">HowlerOps Icon Examples</h2>
      
      {/* Header size icon */}
      <div className="flex items-center space-x-4">
        <HowlerOpsIcon size={24} variant="icon" />
        <span>Header size (24px)</span>
      </div>
      
      {/* Logo size icon */}
      <div className="flex items-center space-x-4">
        <HowlerOpsIcon size={64} variant="logo" />
        <span>Logo size (64px)</span>
      </div>
      
      {/* Large logo */}
      <div className="flex items-center space-x-4">
        <HowlerOpsIcon size={128} variant="logo" />
        <span>Large logo (128px)</span>
      </div>
      
      {/* With custom styling */}
      <div className="flex items-center space-x-4">
        <HowlerOpsIcon 
          size={32} 
          variant="icon" 
          className="hover:opacity-80 transition-opacity cursor-pointer" 
        />
        <span>With hover effects</span>
      </div>
    </div>
  );
};

export default IconExamples;
