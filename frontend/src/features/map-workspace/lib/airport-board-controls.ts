export type AirportOption = {
  icao: string;
  iata: string;
  name: string;
  city: string;
};

export const AIRPORT_OPTIONS: AirportOption[] = [
  { icao: "RJTT", iata: "HND", name: "Tokyo Haneda", city: "Tokyo" },
  { icao: "RJAA", iata: "NRT", name: "Tokyo Narita", city: "Tokyo" },
  { icao: "KJFK", iata: "JFK", name: "John F. Kennedy", city: "New York" },
  { icao: "KLAX", iata: "LAX", name: "Los Angeles", city: "Los Angeles" },
  { icao: "KSFO", iata: "SFO", name: "San Francisco", city: "San Francisco" },
  { icao: "KORD", iata: "ORD", name: "Chicago O'Hare", city: "Chicago" },
  { icao: "EGLL", iata: "LHR", name: "London Heathrow", city: "London" },
  { icao: "LFPG", iata: "CDG", name: "Paris Charles de Gaulle", city: "Paris" },
  { icao: "EDDF", iata: "FRA", name: "Frankfurt", city: "Frankfurt" },
  { icao: "EHAM", iata: "AMS", name: "Amsterdam Schiphol", city: "Amsterdam" },
  { icao: "WSSS", iata: "SIN", name: "Singapore Changi", city: "Singapore" },
  { icao: "VHHH", iata: "HKG", name: "Hong Kong", city: "Hong Kong" },
  { icao: "RKSI", iata: "ICN", name: "Seoul Incheon", city: "Seoul" },
  { icao: "ZBAA", iata: "PEK", name: "Beijing Capital", city: "Beijing" },
  { icao: "YSSY", iata: "SYD", name: "Sydney Kingsford Smith", city: "Sydney" },
];

export function parseDateOnly(value: string): Date | undefined {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (match === null) {
    return undefined;
  }
  const [, yearText, monthText, dayText] = match;
  const year = Number(yearText);
  const month = Number(monthText);
  const day = Number(dayText);
  const date = new Date(year, month - 1, day);

  return date.getFullYear() === year && date.getMonth() === month - 1 && date.getDate() === day
    ? date
    : undefined;
}

export function formatDateOnly(date: Date): string {
  const year = String(date.getFullYear()).padStart(4, "0");
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function airportOptionByCode(code: string): AirportOption | null {
  const normalizedCode = code.trim().toUpperCase();
  return (
    AIRPORT_OPTIONS.find(
      (airport) => airport.icao === normalizedCode || airport.iata === normalizedCode,
    ) ?? null
  );
}

export function airportLabel(airport: AirportOption | null, fallbackCode: string): string {
  if (airport === null) {
    return fallbackCode.trim().toUpperCase();
  }
  return `${airport.name} · ${airport.iata} / ${airport.icao}`;
}
