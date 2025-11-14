import { useEffect, useState } from 'react';
import { X, Download, ExternalLink } from 'lucide-react';
import { Button } from './ui/button';
import { Card } from './ui/card';
import { useUpdateChecker } from '../hooks/use-update-checker';

export function UpdateNotification() {
  const {
    updateInfo,
    isChecking,
    dismissUpdate,
    openDownloadPage,
  } = useUpdateChecker();

  const [isVisible, setIsVisible] = useState(false);

  // Fade in animation when update is available
  useEffect(() => {
    if (updateInfo?.available) {
      setTimeout(() => setIsVisible(true), 100);
    } else {
      setIsVisible(false);
    }
  }, [updateInfo]);

  const handleDismiss = () => {
    setIsVisible(false);
    setTimeout(() => dismissUpdate(), 300); // Wait for fade out animation
  };

  const handleDownload = async () => {
    await openDownloadPage();
    setIsVisible(false);
  };

  if (!updateInfo?.available || isChecking) {
    return null;
  }

  return (
    <div
      className={`fixed top-4 right-4 z-50 transition-all duration-300 ${
        isVisible ? 'opacity-100 translate-y-0' : 'opacity-0 -translate-y-4'
      }`}
    >
      <Card className="w-96 shadow-lg border-2 border-primary/20 bg-card/95 backdrop-blur-sm">
        <div className="p-4 space-y-3">
          {/* Header */}
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-2">
              <div className="p-2 rounded-full bg-primary/10">
                <Download className="h-4 w-4 text-primary" />
              </div>
              <div>
                <h3 className="font-semibold text-sm">Update Available</h3>
                <p className="text-xs text-muted-foreground">
                  Version {updateInfo.latestVersion}
                </p>
              </div>
            </div>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6 -mt-1 -mr-1"
              onClick={handleDismiss}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>

          {/* Release Notes Preview */}
          {updateInfo.releaseNotes && (
            <div className="text-xs text-muted-foreground max-h-24 overflow-y-auto">
              <p className="line-clamp-4">
                {updateInfo.releaseNotes.split('\n')[0] || 'New features and improvements'}
              </p>
            </div>
          )}

          {/* Actions */}
          <div className="flex gap-2">
            <Button
              size="sm"
              className="flex-1 gap-2"
              onClick={handleDownload}
            >
              <ExternalLink className="h-3 w-3" />
              Download Update
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleDismiss}
            >
              Later
            </Button>
          </div>

          {/* Current Version Info */}
          <div className="text-xs text-muted-foreground pt-2 border-t">
            Current version: {updateInfo.currentVersion}
          </div>
        </div>
      </Card>
    </div>
  );
}
