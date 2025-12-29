import { Wallet, LineChart, ShieldCheck, Building2, Layers, Clock } from "lucide-react";

const features = [
  {
    icon: Wallet,
    title: "Structured Lending",
    description: "Earn competitive yields by lending your digital assets through our institutional-grade platform with full transparency.",
  },
  {
    icon: LineChart,
    title: "Market Arbitrage",
    description: "Benefit from sophisticated trading strategies that capture cross-exchange spreads, basis trades, and liquidity premiums.",
  },
  {
    icon: ShieldCheck,
    title: "Cold Storage Security",
    description: "All client assets stored in multi-signature or MPC cold wallets, completely segregated from operational funds.",
  },
  {
    icon: Building2,
    title: "Institutional Framework",
    description: "Enterprise-grade risk management with real-time monitoring, credit assessment, and comprehensive stress testing.",
  },
  {
    icon: Layers,
    title: "Structured Products",
    description: "Access sophisticated investment vehicles including fixed-income notes and strategic yield enhancement products.",
  },
  {
    icon: Clock,
    title: "Flexible Terms",
    description: "Choose from multiple lending durations and yield tiers to match your investment timeline and risk appetite.",
  },
];

const Features = () => {
  return (
    <section id="products" className="py-24 relative">
      <div className="container mx-auto px-6">
        {/* Section Header */}
        <div className="text-center max-w-2xl mx-auto mb-16">
          <span className="text-primary text-sm font-medium tracking-wider uppercase mb-4 block">
            Core Capabilities
          </span>
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-foreground mb-6">
            Institutional Excellence
          </h2>
          <p className="text-muted-foreground text-lg">
            A comprehensive suite of digital asset services designed for sophisticated investors 
            seeking premium returns with institutional-grade security.
          </p>
        </div>

        {/* Features Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {features.map((feature, index) => (
            <div
              key={feature.title}
              className="group p-8 rounded-2xl glass hover:bg-card/80 transition-all duration-300 hover:border-primary/30"
              style={{ animationDelay: `${index * 0.1}s` }}
            >
              <div className="w-12 h-12 rounded-xl bg-primary/10 flex items-center justify-center mb-6 group-hover:bg-primary/20 transition-colors">
                <feature.icon className="w-6 h-6 text-primary" />
              </div>
              <h3 className="text-xl font-semibold text-foreground mb-3">
                {feature.title}
              </h3>
              <p className="text-muted-foreground leading-relaxed">
                {feature.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default Features;
