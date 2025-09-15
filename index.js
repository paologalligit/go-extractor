const fs = require('fs');
const path = require('path');

// Use the already existing showings.json file
const showingsFile = 'showings.json';

const data = JSON.parse(fs.readFileSync(path.join(__dirname, showingsFile), 'utf8'));

function getAllSessionsSortedBySeatsDesc(data) {
  const sessions = [];
  data.forEach(movie => {
    if (!movie.showingGroups) return;
    movie.showingGroups.forEach(group => {
      (group.sessions || []).forEach(session => {
        sessions.push({
          movie: movie.movie || movie.filmTitle || '',
          cinemaName: movie.cinemaName || '',
          startTime: session.startTime || '',
          seats: session.seats || 0,
        });
      });
    });
  });
  // Sort descending by seats
  sessions.sort((a, b) => b.seats - a.seats);
  return sessions;
}

// Example usage: print top 10 sessions by seats
const sortedSessions = getAllSessionsSortedBySeatsDesc(data);
console.log('Top 10 sessions by seats:');
console.log(sortedSessions.slice(0, 10));