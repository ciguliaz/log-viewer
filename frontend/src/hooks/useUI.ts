import { useState, useEffect } from 'react';
import { CheckForUpdate, ApplyUpdate, GetVersion } from '../../wailsjs/go/main/App';
import { Quit } from '../../wailsjs/runtime/runtime';
import { ToastMessage } from '../components/Toast';

export function useUI() {
    // Toast
    const [toast, setToast] = useState<ToastMessage | null>(null);
    const showToast = (message: string, type: 'success' | 'error' = 'success') => {
        setToast({ message, type });
        setTimeout(() => setToast(null), 3000);
    };

    // Update State
    const [updateInfo, setUpdateInfo] = useState<{available: boolean, version: string, releaseNotes: string} | null>(null);
    const [showUpdateModal, setShowUpdateModal] = useState(false);
    const [isUpdating, setIsUpdating] = useState(false);
    const [appVersion, setAppVersion] = useState<string>('');
    const [isCheckingUpdate, setIsCheckingUpdate] = useState(false);

    // Feedback State
    const [showFeedbackModal, setShowFeedbackModal] = useState(false);
    const [feedbackType, setFeedbackType] = useState<'Bug' | 'Feature'>('Bug');
    const [feedbackText, setFeedbackText] = useState('');
    const [isSubmittingFeedback, setIsSubmittingFeedback] = useState(false);

    const checkUpdate = async (manual: boolean = false) => {
        if (isCheckingUpdate) return;
        if (manual) setIsCheckingUpdate(true);
        try {
            const info = await CheckForUpdate();
            if (info && info.available) {
                setUpdateInfo(info);
                if (manual) setShowUpdateModal(true);
            } else if (manual) {
                showToast("You are on the latest version.");
            }
        } catch (e) {
            console.error("Failed to check for updates:", e);
            if (manual) showToast("Failed to check for updates.", "error");
        } finally {
            if (manual) setIsCheckingUpdate(false);
        }
    };

    useEffect(() => {
        GetVersion().then(v => setAppVersion(v)).catch(() => {});
        setTimeout(() => checkUpdate(false), 2000);
        const intervalId = setInterval(() => checkUpdate(false), 6 * 60 * 60 * 1000);
        return () => clearInterval(intervalId);
    }, []);

    const handleApplyUpdate = async () => {
        setIsUpdating(true);
        try {
            // @ts-ignore
            await ApplyUpdate();
            showToast("Update successful! Please restart the application.", "success");
            setTimeout(() => Quit(), 1500);
        } catch (e) {
            console.error("Update failed:", e);
            showToast("Update failed: " + e, "error");
            setIsUpdating(false);
        }
    };

    const handleSubmitFeedback = async () => {
        if (!feedbackText.trim()) return;
        setIsSubmittingFeedback(true);
        try {
            const response = await fetch("https://hook.eu1.make.com/kia7d51g84oar77mnhywam9ujzvivwvx", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ type: feedbackType, message: feedbackText, version: appVersion || 'unknown' }),
            });
            if (response.ok) {
                showToast("Thank you! Your feedback has been submitted successfully.", "success");
                setShowFeedbackModal(false);
                setFeedbackText('');
            } else {
                showToast("Failed to submit feedback. Please try again later.", "error");
            }
        } catch (e) {
            console.error("Feedback error:", e);
            showToast("Failed to submit feedback. Please check your internet connection.", "error");
        } finally {
            setIsSubmittingFeedback(false);
        }
    };

    return {
        toast, showToast,
        updateInfo, showUpdateModal, setShowUpdateModal, isUpdating, handleApplyUpdate, checkUpdate, appVersion, isCheckingUpdate,
        showFeedbackModal, setShowFeedbackModal, feedbackType, setFeedbackType, feedbackText, setFeedbackText, isSubmittingFeedback, handleSubmitFeedback
    };
}
