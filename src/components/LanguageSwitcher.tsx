import { useTranslation } from 'react-i18next';
import { Button } from '@/components/ui/button';
import { Globe } from 'lucide-react';

const LanguageSwitcher = () => {
  const { i18n } = useTranslation();

  const toggleLanguage = () => {
    const newLng = i18n.language === 'en' ? 'zh' : 'en';
    console.log('Switching to:', newLng);
    i18n.changeLanguage(newLng).then(() => {
      console.log('Language changed to:', i18n.language);
    });
  };

  const displayText = i18n.language === 'en' ? '中' : 'EN';

  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={toggleLanguage}
      className="flex items-center gap-2"
      title={i18n.language === 'en' ? '切换到中文' : 'Switch to English'}
    >
      <Globe size={16} />
      <span className="font-medium">{displayText}</span>
    </Button>
  );
};

export default LanguageSwitcher;
