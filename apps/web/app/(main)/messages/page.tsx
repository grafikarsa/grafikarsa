'use client';

import { useState } from 'react';
import { ChatWindow } from '@/components/dm/chat-window';
import { ConversationList } from '@/components/dm/conversation-list';
import { NewMessageModal } from '@/components/dm/new-message-modal';
import { DMSettingsModal } from '@/components/dm/dm-settings-modal';
import { Button } from '@/components/ui/button';
import { SquarePen, Settings } from 'lucide-react';
import { useAuthStore } from '@/lib/stores/auth-store';

export default function MessagesPage() {
    const { user } = useAuthStore();
    const [newItemOpen, setNewItemOpen] = useState(false);
    const [settingsOpen, setSettingsOpen] = useState(false);

    return (
        <div className="h-[calc(100vh-80px)] max-w-6xl mx-auto p-0 md:p-4 flex">
            {/* Main Container - Instagram Style Border */}
            <div className="flex-1 flex bg-background border rounded-none md:rounded-xl overflow-hidden shadow-sm">

                {/* Sidebar List (35% width, min 300px) */}
                <div className="w-full md:w-[350px] flex flex-col border-r bg-background z-10">
                    {/* Header */}
                    <div className="flex items-center justify-between p-5 border-b h-[72px]">
                        <div className="flex items-center gap-2">
                            <h1 className="font-bold text-xl">{user?.username}</h1>
                            <Button variant="ghost" size="icon" className="h-6 w-6" onClick={() => setSettingsOpen(true)}>
                                <Settings className="h-4 w-4" />
                            </Button>
                        </div>
                        <Button variant="ghost" size="icon" onClick={() => setNewItemOpen(true)}>
                            <SquarePen className="h-6 w-6" />
                        </Button>
                    </div>

                    {/* Conversation List */}
                    <div className="flex-1 overflow-hidden">
                        <ConversationList />
                    </div>
                </div>

                {/* Main Chat Area (Remaining width) */}
                <div className="hidden md:flex flex-1 flex-col bg-background relative">
                    <ChatWindow onOpenInfo={() => setSettingsOpen(true)} />
                </div>
            </div>

            {/* Modals */}
            <NewMessageModal open={newItemOpen} onOpenChange={setNewItemOpen} />
            <DMSettingsModal open={settingsOpen} onOpenChange={setSettingsOpen} />
        </div>
    );
}

