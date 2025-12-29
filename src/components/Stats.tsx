const stats = [
  { value: "$100M+", label: "Target AUM", sublabel: "Year 1" },
  { value: "8-12%", label: "Target APY", sublabel: "Risk-adjusted" },
  { value: "99.95%", label: "Uptime SLA", sublabel: "Platform availability" },
  { value: "100%", label: "Asset Segregation", sublabel: "Cold storage" },
];

const Stats = () => {
  return (
    <section className="py-20 relative">
      {/* Background Accent */}
      <div className="absolute inset-0 bg-gradient-to-b from-transparent via-primary/[0.02] to-transparent" />
      
      <div className="container mx-auto px-6 relative z-10">
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-8">
          {stats.map((stat, index) => (
            <div
              key={stat.label}
              className="text-center p-6 rounded-2xl glass"
              style={{ animationDelay: `${index * 0.1}s` }}
            >
              <div className="text-3xl sm:text-4xl lg:text-5xl font-bold text-primary mb-2">
                {stat.value}
              </div>
              <div className="text-foreground font-medium mb-1">
                {stat.label}
              </div>
              <div className="text-sm text-muted-foreground">
                {stat.sublabel}
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

export default Stats;
