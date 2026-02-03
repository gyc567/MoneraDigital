import { Wallet, LineChart, ShieldCheck, Building2, Layers, Clock } from "lucide-react";
import { useTranslation } from "react-i18next";

const iconMap = {
  "Structured Lending": Wallet,
  "Market Arbitrage": LineChart,
  "Cold Storage Security": ShieldCheck,
  "Institutional Framework": Building2,
  "Structured Products": Layers,
  "Flexible Terms": Clock,
};

const Features = () => {
  const { t } = useTranslation();

  const rawFeatures = t("features.items", { returnObjects: true });
  const features = Array.isArray(rawFeatures) ? rawFeatures : [];

  return (
    <section id="products" className="py-24 relative">
      <div className="container mx-auto px-6">
        {/* Section Header */}
        <div className="text-center max-w-2xl mx-auto mb-16">
          <span className="text-primary text-sm font-medium tracking-wider uppercase mb-4 block">
            {t("features.sectionLabel")}
          </span>
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-foreground mb-6">
            {t("features.title")}
          </h2>
          <p className="text-muted-foreground text-lg">
            {t("features.description")}
          </p>
        </div>

        {/* Features Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {features.map((feature, index) => {
            const iconKey = feature.title as keyof typeof iconMap;
            const Icon = iconMap[iconKey] || Wallet;

            return (
              <div
                key={feature.title}
                className="group p-8 rounded-2xl glass hover:bg-card/80 transition-all duration-300 hover:border-primary/30"
                style={{ animationDelay: `${index * 0.1}s` }}
              >
                <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center mb-6 group-hover:bg-primary/20 transition-colors">
                  <Icon className="w-6 h-6 text-primary" />
                </div>
                <h3 className="text-xl font-semibold text-foreground mb-3">
                  {feature.title}
                </h3>
                <p className="text-muted-foreground leading-relaxed">
                  {feature.description}
                </p>
              </div>
            );
          })}
        </div>
      </div>
    </section>
  );
};

export default Features;
