import { useCallback,useEffect, useState } from 'react';

import { CheckForUpdates, GetCurrentVersion,OpenDownloadPage } from '../../wailsjs/go/main/App';

export interface UpdateInfo {
  available: boolean;
  currentVersion: string;
  latestVersion: string;
  downloadUrl: string;
  releaseNotes: string;
  publishedAt: string;
}

export interface UseUpdateCheckerReturn {
  updateInfo: UpdateInfo | null;
  isChecking: boolean;
  error: string | null;
  checkForUpdates: () => Promise<void>;
  dismissUpdate: () => void;
  openDownloadPage: () => Promise<void>;
  currentVersion: string;
}

const UPDATE_CHECK_INTERVAL = 24 * 60 * 60 * 1000; // 24 hours
const DISMISSED_UPDATE_KEY = 'howlerops_dismissed_update';

export function useUpdateChecker(): UseUpdateCheckerReturn {
  const [updateInfo, setUpdateInfo] = useState<UpdateInfo | null>(null);
  const [isChecking, setIsChecking] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [currentVersion, setCurrentVersion] = useState<string>('');

  // Check if an update was dismissed
  const isDismissed = useCallback((version: string): boolean => {
    const dismissed = localStorage.getItem(DISMISSED_UPDATE_KEY);
    return dismissed === version;
  }, []);

  // Check for updates
  const checkForUpdates = useCallback(async () => {
    setIsChecking(true);
    setError(null);

    try {
      const info = await CheckForUpdates();

      // Don't show notification if this version was already dismissed
      if (info.available && isDismissed(info.latestVersion)) {
        setUpdateInfo(null);
      } else {
        setUpdateInfo(info);
      }
    } catch (err) {
      console.error('Failed to check for updates:', err);
      setError(err instanceof Error ? err.message : 'Failed to check for updates');
      setUpdateInfo(null);
    } finally {
      setIsChecking(false);
    }
  }, [isDismissed]);

  // Dismiss update notification
  const dismissUpdate = useCallback(() => {
    if (updateInfo?.latestVersion) {
      localStorage.setItem(DISMISSED_UPDATE_KEY, updateInfo.latestVersion);
    }
    setUpdateInfo(null);
  }, [updateInfo]);

  // Open download page
  const openDownloadPage = useCallback(async () => {
    try {
      await OpenDownloadPage();
      // After opening download page, dismiss the notification
      dismissUpdate();
    } catch (err) {
      console.error('Failed to open download page:', err);
      setError(err instanceof Error ? err.message : 'Failed to open download page');
    }
  }, [dismissUpdate]);

  // Get current version on mount
  useEffect(() => {
    const fetchCurrentVersion = async () => {
      try {
        const version = await GetCurrentVersion();
        setCurrentVersion(version);
      } catch (err) {
        console.error('Failed to get current version:', err);
      }
    };

    fetchCurrentVersion();
  }, []);

  // Check for updates on mount and periodically
  useEffect(() => {
    // Initial check after 5 seconds (give app time to load)
    const initialTimeout = setTimeout(() => {
      checkForUpdates();
    }, 5000);

    // Periodic checks
    const interval = setInterval(() => {
      checkForUpdates();
    }, UPDATE_CHECK_INTERVAL);

    return () => {
      clearTimeout(initialTimeout);
      clearInterval(interval);
    };
  }, [checkForUpdates]);

  return {
    updateInfo,
    isChecking,
    error,
    checkForUpdates,
    dismissUpdate,
    openDownloadPage,
    currentVersion,
  };
}
