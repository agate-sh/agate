export function generateBranchName(): string {
  const adjectives = [
    'quick',
    'bright',
    'swift',
    'clever',
    'bold',
    'neat',
    'clean',
    'smooth',
    'sharp',
    'cool',
  ];

  const nouns = [
    'fix',
    'update',
    'patch',
    'change',
    'work',
    'task',
    'feature',
    'test',
    'demo',
    'trial',
  ];

  const adj = adjectives[Math.floor(Math.random() * adjectives.length)];
  const noun = nouns[Math.floor(Math.random() * nouns.length)];

  return `${adj}-${noun}`;
}
