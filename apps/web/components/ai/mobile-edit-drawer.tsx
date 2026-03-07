'use client';

import { Button } from '@/components/ui/button';
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from '@/components/ui/drawer';
import { Edit3 } from 'lucide-react';

interface MobileEditDrawerProps {
  children: React.ReactNode;
  onSave?: () => void;
}

export function MobileEditDrawer({ children, onSave }: MobileEditDrawerProps) {
  return (
    <Drawer>
      <DrawerTrigger asChild>
        <Button variant="outline" size="sm" className="gap-2 md:hidden">
          <Edit3 className="h-4 w-4" />
          Edit Preferensi
        </Button>
      </DrawerTrigger>
      <DrawerContent className="max-h-[85vh]">
        <DrawerHeader>
          <DrawerTitle>Edit Preferensi</DrawerTitle>
          <DrawerDescription>
            Ubah preferensi untuk generate ide baru
          </DrawerDescription>
        </DrawerHeader>
        <div className="overflow-y-auto px-4 pb-4">
          {children}
        </div>
        <DrawerFooter>
          <DrawerClose asChild>
            <Button onClick={onSave} className="w-full">
              Simpan & Generate Ulang
            </Button>
          </DrawerClose>
          <DrawerClose asChild>
            <Button variant="outline" className="w-full">
              Batal
            </Button>
          </DrawerClose>
        </DrawerFooter>
      </DrawerContent>
    </Drawer>
  );
}
